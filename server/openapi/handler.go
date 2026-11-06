package openapi

import (
	"go/ast"
	"go/token"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// handlerAnalysisState 保存 handler 函数分析过程中的状态
type handlerAnalysisState struct {
	varTypes   map[string]string // 变量名 → 类型名（如 "resp" → "account.LoginResponse"）
	localTypes map[string]string // 本地类型别名
	filename   string            // 当前分析的文件
	helpers    map[string]bool   // 已跟进过的同文件辅助函数（防递归）
}

func (g *generator) analyzeHandler(r *RouteDef) {
	// 内联 handler：直接从路由定义所在文件分析闭包体
	if fn, ok := g.inlineFuncs[r.Handler]; ok {
		f := g.parseFile(r.HandlerFile)
		var localTypes map[string]string
		if f != nil {
			localTypes = g.collectLocalTypes(f)
		}
		g.analyzeFuncBody(r, fn.Body, localTypes, r.HandlerFile)
		return
	}

	// 找到 handler 对应的 ctrl_*.go 文件
	_, funcName, ok := strings.Cut(r.Handler, ".")
	if !ok {
		return
	}

	ctrlDir := filepath.Join(g.root, g.cfg.CtrlDir)

	// 缓存目录条目，避免每个路由重复读取
	if g.ctrlEntries == nil {
		var err error
		g.ctrlEntries, err = os.ReadDir(ctrlDir)
		if err != nil {
			return
		}
	}

	for _, entry := range g.ctrlEntries {
		if !strings.HasPrefix(entry.Name(), g.cfg.CtrlPrefix) || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filename := filepath.Join(ctrlDir, entry.Name())
		f := g.parseFile(filename)
		if f == nil {
			continue
		}

		// 查找 handler 函数
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != funcName || fd.Body == nil {
				continue
			}
			// 验证 receiver
			if !isAppReceiver(fd) {
				continue
			}

			// 收集本地类型别名（如 filerPathQuery, cronJobEnableReq）
			localTypes := g.collectLocalTypes(f)

			// 分析函数体
			g.analyzeFuncBody(r, fd.Body, localTypes, filename)
			return
		}
	}
}

// ─── 辅助函数 ───

func (g *generator) analyzeFuncBody(r *RouteDef, body *ast.BlockStmt, localTypes map[string]string, filename string) {
	state := &handlerAnalysisState{
		varTypes:   make(map[string]string),
		localTypes: localTypes,
		filename:   filename,
		helpers:    make(map[string]bool),
	}

	ast.Inspect(body, func(n ast.Node) bool {
		g.analyzeNode(n, r, state)
		return true
	})
}

// analyzeNode 按节点类型分发到各分析器
func (g *generator) analyzeNode(n ast.Node, r *RouteDef, state *handlerAnalysisState) {
	switch stmt := n.(type) {
	case *ast.AssignStmt:
		g.analyzeAssignStmt(stmt, state)
	case *ast.DeclStmt:
		g.analyzeDeclStmt(stmt, r, state)
	case *ast.CallExpr:
		g.analyzeCallExpr(stmt, r, state)
	case *ast.IndexExpr:
		analyzeMultipartIndex(stmt, r)
	}
}

// analyzeAssignStmt 处理赋值语句，跟踪变量类型
func (g *generator) analyzeAssignStmt(stmt *ast.AssignStmt, state *handlerAnalysisState) {
	if len(stmt.Rhs) != 1 {
		return
	}

	// 处理 := 短变量声明
	if stmt.Tok == token.DEFINE {
		// 收集左侧变量名
		var lhsNames []string
		for _, lhs := range stmt.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				lhsNames = append(lhsNames, ident.Name)
			}
		}

		// 情况 1: resp := pkg.Type{} 复合字面量
		if comp, ok := stmt.Rhs[0].(*ast.CompositeLit); ok {
			typeName := typeExprToString(comp.Type)
			if typeName != "" && typeName != "struct" && len(lhsNames) > 0 {
				state.varTypes[lhsNames[0]] = typeName
			}
			return
		}

		// 情况 1b: resp := &pkg.Type{} 取地址的复合字面量
		if unary, ok := stmt.Rhs[0].(*ast.UnaryExpr); ok && unary.Op == token.AND {
			if comp, ok := unary.X.(*ast.CompositeLit); ok {
				typeName := typeExprToString(comp.Type)
				if typeName != "" && typeName != "struct" && len(lhsNames) > 0 {
					state.varTypes[lhsNames[0]] = typeName
				}
				return
			}
		}

		// 情况 2: resp, err := app.svc.Method()
		if callExpr, ok := stmt.Rhs[0].(*ast.CallExpr); ok && len(lhsNames) >= 1 {
			retType := g.resolveCallReturnType(callExpr, state.filename)
			if retType != "" {
				state.varTypes[lhsNames[0]] = retType
			}
			return
		}
	}

	// 处理普通赋值 =
	// 如: result = app.svc.Method()
	if len(stmt.Lhs) == 1 {
		if ident, ok := stmt.Lhs[0].(*ast.Ident); ok {
			if callExpr, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
				retType := g.resolveCallReturnType(callExpr, state.filename)
				if retType != "" {
					state.varTypes[ident.Name] = retType
				}
			}
		}
	}
}

// analyzeDeclStmt 处理声明语句，跟踪变量类型
func (g *generator) analyzeDeclStmt(stmt *ast.DeclStmt, r *RouteDef, state *handlerAnalysisState) {
	gd, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return
	}

	for _, spec := range gd.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) == 1 {
			varName := vs.Names[0].Name
			if vs.Type == nil {
				continue // 类型推断形式（var x = ...），跳过
			}
			typeName := typeExprToString(vs.Type)
			if sel, ok := vs.Type.(*ast.SelectorExpr); ok {
				typeName = fullSelName(sel)
			}
			state.varTypes[varName] = typeName

			// 内联 struct 类型也记录
			if st, ok := vs.Type.(*ast.StructType); ok {
				schema := inlineStructToSchema(st)
				if schema != nil {
					cacheKey := r.Module + "." + extractHandlerName(r.Handler)
					schema.TypeName = extractHandlerName(r.Handler)
					schema.PkgName = r.Module
					g.structCache[cacheKey] = schema
					state.varTypes[varName] = cacheKey
					g.resolveNestedTypesInSchema(schema, "")
				}
			}
		}
	}
}

// analyzeCallExpr 处理函数调用表达式
func (g *generator) analyzeCallExpr(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	// 处理直接函数调用（如 respondSuccess、respondResult、respondResultMsg、respondError）
	if ident, ok := stmt.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "respondSuccess":
			if len(stmt.Args) >= 3 {
				g.analyzeRespondSuccess(stmt, r, state)
			}
			return
		case "respondResult":
			// respondResult(c, data, err)：data 位于第 2 个参数
			if len(stmt.Args) >= 3 {
				g.analyzeRespondData(stmt.Args[1], r, state)
			}
			return
		case "respondResultMsg":
			// respondResultMsg(c, msg, data, err)：data 位于第 3 个参数
			if len(stmt.Args) >= 4 {
				g.analyzeRespondData(stmt.Args[2], r, state)
			}
			return
		case "respondError":
			// 错误处理，无需特殊处理
			return
		default:
			g.analyzeHelperFunc(r, state, ident.Name)
			return
		}
	}

	sel, ok := stmt.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	switch sel.Sel.Name {
	case "ShouldBindJSON":
		g.analyzeShouldBindJSON(stmt, r, state)
	case "ShouldBindQuery":
		g.analyzeShouldBindQuery(stmt, r, state)
	case "Query":
		extractQueryParam(stmt, r, "string", false)
	case "DefaultQuery":
		extractDefaultQueryParam(stmt, r)
	case "Param":
		extractParamCall(stmt, r)
	case "PostForm", "GetPostForm":
		r.FormData = true
		extractFormField(stmt, r, "string")
	case "FormFile":
		r.FormData = true
		extractFormField(stmt, r, "file")
	case "ParseMultipartForm":
		r.FormData = true
	case "NewEventWriter":
		r.IsSSE = true
	case "Data":
		analyzeTextResponse(stmt, r)
	}

	// 检测 WebSocket：既支持直接的 wsConfig.Handler，也支持 serveWebSocket 封装
	if sel.Sel.Name == "serveWebSocket" {
		r.IsWS = true
	}
	if sel2, ok2 := sel.X.(*ast.SelectorExpr); ok2 && sel2.Sel.Name == "wsConfig" {
		r.IsWS = true
	}
}

// analyzeHelperFunc 跟进同文件辅助函数，分析其中的绑定与参数提取调用。
// 例如 compose 部署把 JSON/multipart 绑定抽到了 bindComposeDeployRequest，
// 不跟进会导致生成的 OpenAPI 缺失 requestBody。
func (g *generator) analyzeHelperFunc(r *RouteDef, state *handlerAnalysisState, funcName string) {
	if state.helpers[funcName] {
		return
	}
	state.helpers[funcName] = true

	f := g.parseFile(state.filename)
	if f == nil {
		return
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName || fd.Body == nil {
			continue
		}
		sub := &handlerAnalysisState{
			varTypes:   make(map[string]string),
			localTypes: state.localTypes,
			filename:   state.filename,
			helpers:    state.helpers,
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			g.analyzeNode(n, r, sub)
			return true
		})
		return
	}
}

// analyzeShouldBindJSON 分析 ShouldBindJSON 调用
func (g *generator) analyzeShouldBindJSON(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	if len(stmt.Args) < 1 {
		return
	}
	var varName string
	if unary, ok := stmt.Args[0].(*ast.UnaryExpr); ok && unary.Op == token.AND {
		if ident, ok := unary.X.(*ast.Ident); ok {
			varName = ident.Name
		}
	}
	if varName == "" {
		return
	}
	if typeName, ok := state.varTypes[varName]; ok && r.JSONBody == nil {
		schema := g.resolveStructSchema(typeName, state.localTypes, state.filename)
		if schema != nil {
			r.JSONBody = schema
		}
	}
}

// analyzeShouldBindQuery 分析 ShouldBindQuery 调用
func (g *generator) analyzeShouldBindQuery(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	if len(stmt.Args) < 1 {
		return
	}
	var varName string
	if unary, ok := stmt.Args[0].(*ast.UnaryExpr); ok && unary.Op == token.AND {
		if ident, ok := unary.X.(*ast.Ident); ok {
			varName = ident.Name
		}
	}
	if varName == "" {
		return
	}
	if typeName, ok := state.varTypes[varName]; ok && r.QueryType == nil {
		schema := g.resolveStructSchema(typeName, state.localTypes, state.filename)
		if schema != nil {
			r.QueryType = schema
		}
	}
}

// analyzeRespondSuccess 分析 respondSuccess(c, msg, data) 调用
func (g *generator) analyzeRespondSuccess(call *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	if len(call.Args) < 3 {
		return
	}
	g.analyzeRespondData(call.Args[2], r, state)
}

// analyzeRespondData 根据响应 data 参数表达式推断 payload 类型，
// 供 respondSuccess(c, msg, data) 与 respondResult(c, data, err) 共用
func (g *generator) analyzeRespondData(dataArg ast.Expr, r *RouteDef, state *handlerAnalysisState) {
	// 情况 1: nil → 无 payload
	if ident, ok := dataArg.(*ast.Ident); ok && ident.Name == "nil" {
		return
	}

	// 情况 2: 变量引用，如 resp → 查找变量类型
	if ident, ok := dataArg.(*ast.Ident); ok {
		typeName, ok := state.varTypes[ident.Name]
		if ok && typeName != "" {
			r.ResponseBodyType = typeName
			// 确保类型 schema 被解析到缓存
			g.resolveStructSchema(typeName, state.localTypes, state.filename)
		}
		return
	}

	// 情况 3: 复合字面量，如 svcType.RespType{...}
	if comp, ok := dataArg.(*ast.CompositeLit); ok {
		typeName := typeExprToString(comp.Type)
		if typeName != "" && typeName != "struct" {
			r.ResponseBodyType = typeName
		}
		return
	}

	// 情况 4: 函数调用返回值，如 app.svc.Method()
	if callExpr, ok := dataArg.(*ast.CallExpr); ok {
		retType := g.resolveCallReturnType(callExpr, state.filename)
		if retType != "" {
			r.ResponseBodyType = retType
		}
		return
	}
}

// analyzeTextResponse 识别 HTTP 200 的原始文本响应。
func analyzeTextResponse(stmt *ast.CallExpr, r *RouteDef) {
	if len(stmt.Args) < 3 {
		return
	}
	statusOK := false
	switch status := stmt.Args[0].(type) {
	case *ast.SelectorExpr:
		pkg, ok := status.X.(*ast.Ident)
		statusOK = ok && pkg.Name == "http" && status.Sel.Name == "StatusOK"
	case *ast.BasicLit:
		code, err := strconv.Atoi(status.Value)
		statusOK = err == nil && code == 200
	}
	if !statusOK {
		return
	}
	contentType := stringConstant(stmt.Args[1], 0)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == "text/plain" {
		r.RawContentType = mediaType
	}
}

// extractQueryParam 提取 Query/PostForm 参数
func extractQueryParam(stmt *ast.CallExpr, r *RouteDef, typ string, required bool) {
	if len(stmt.Args) >= 1 {
		if lit, ok := stmt.Args[0].(*ast.BasicLit); ok {
			name := strings.Trim(lit.Value, `"`)
			r.QueryParams = append(r.QueryParams, ParamDef{
				Name: name, Type: typ, Required: required,
			})
		}
	}
}

// extractDefaultQueryParam 提取 DefaultQuery 参数
func extractDefaultQueryParam(stmt *ast.CallExpr, r *RouteDef) {
	if len(stmt.Args) >= 2 {
		if lit, ok := stmt.Args[0].(*ast.BasicLit); ok {
			name := strings.Trim(lit.Value, `"`)
			typ := "string"
			if def, ok := stmt.Args[1].(*ast.BasicLit); ok {
				if def.Kind == token.INT {
					typ = "integer"
				}
			}
			r.QueryParams = append(r.QueryParams, ParamDef{
				Name: name, Type: typ, Required: false,
			})
		}
	}
}

// extractParamCall 提取 Param 参数
func extractParamCall(stmt *ast.CallExpr, r *RouteDef) {
	if len(stmt.Args) >= 1 {
		if lit, ok := stmt.Args[0].(*ast.BasicLit); ok {
			name := strings.Trim(lit.Value, `"`)
			if !hasParam(r.PathParams, name) {
				r.PathParams = append(r.PathParams, ParamDef{
					Name: name, Type: "string", Required: true,
				})
			}
		}
	}
}

// extractFormField 提取 multipart 表单字段（PostForm/GetPostForm/FormFile）
func extractFormField(stmt *ast.CallExpr, r *RouteDef, typ string) {
	if len(stmt.Args) < 1 {
		return
	}
	if lit, ok := stmt.Args[0].(*ast.BasicLit); ok {
		name := strings.Trim(lit.Value, `"`)
		if !hasParam(r.FormFields, name) {
			r.FormFields = append(r.FormFields, ParamDef{Name: name, Type: typ})
		}
	}
}

// analyzeMultipartIndex 提取 c.Request.MultipartForm.File["file"] / .Value["key"] 直取形式的表单字段
func analyzeMultipartIndex(expr *ast.IndexExpr, r *RouteDef) {
	sel, ok := expr.X.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "File" && sel.Sel.Name != "Value") {
		return
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "MultipartForm" {
		return
	}
	lit, ok := expr.Index.(*ast.BasicLit)
	if !ok {
		return
	}
	r.FormData = true
	typ := "string"
	if sel.Sel.Name == "File" {
		typ = "files" // MultipartForm.File 是多文件切片
	}
	name := strings.Trim(lit.Value, `"`)
	if !hasParam(r.FormFields, name) {
		r.FormFields = append(r.FormFields, ParamDef{Name: name, Type: typ})
	}
}

// isAppReceiver 检查函数是否有 (*App) receiver
func isAppReceiver(fd *ast.FuncDecl) bool {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return false
	}
	recvType := fd.Recv.List[0].Type
	star, ok := recvType.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "App"
}

// extractHandlerName 从 Handler 字符串提取函数名
func extractHandlerName(handler string) string {
	_, name, _ := strings.Cut(handler, ".")
	return name
}

// hasParam 检查参数列表中是否已包含某个名称
func hasParam(params []ParamDef, name string) bool {
	for _, p := range params {
		if p.Name == name {
			return true
		}
	}
	return false
}

func stringConstant(expr ast.Expr, depth int) string {
	if depth > 16 {
		return ""
	}
	switch value := expr.(type) {
	case *ast.BasicLit:
		if value.Kind == token.STRING {
			text, err := strconv.Unquote(value.Value)
			if err == nil {
				return text
			}
		}
	case *ast.ParenExpr:
		return stringConstant(value.X, depth+1)
	case *ast.Ident:
		if value.Obj == nil || value.Obj.Kind != ast.Con {
			return ""
		}
		if spec, ok := value.Obj.Decl.(*ast.ValueSpec); ok {
			for i, name := range spec.Names {
				if name.Name == value.Name && i < len(spec.Values) {
					return stringConstant(spec.Values[i], depth+1)
				}
			}
		}
	}
	return ""
}
