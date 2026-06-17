// Command openapi-gen 从 isrvd 源码自动生成 OpenAPI 3.0 文档。
//
// 用法:
//
//	go run ./cmd/openapi-gen/  # 生成到 public/openapi/apis.json
//	go run ./cmd/openapi-gen/ -o openapi.json  # 生成到指定路径
//
// 工作原理:
//  1. 解析 internal/server/ctrl_*.go 中的 define*Routes() 方法，提取所有路由定义
//  2. 分析 handler 函数体，提取 ShouldBindJSON/ShouldBindQuery 引用的请求结构体
//  3. 从 internal/service/ 中解析结构体定义及其 json tag
//  4. 提取 c.Param / c.Query / c.DefaultQuery 调用，推断路径参数和查询参数
//  5. 组装成标准 OpenAPI 3.0.3 JSON 输出
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// ─── OpenAPI 输出类型 ────────────────────────────────

type OpenAPI struct {
	OpenAPI    string                    `json:"openapi"`
	Info       OpenAPIInfo               `json:"info"`
	Servers    []OpenAPIServer           `json:"servers"`
	Paths      map[string]map[string]any `json:"paths"`
	Components OpenAPIComponents         `json:"components"`
}

type OpenAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIComponents struct {
	Schemas         map[string]any `json:"schemas"`
	SecuritySchemes map[string]any `json:"securitySchemes"`
}

// ─── 路由中间表示 ────────────────────────────────────

type RouteDef struct {
	Method  string // GET/POST/PUT/PATCH/DELETE/ANY
	Path    string // /overview/probe
	Label   string // 中文描述
	Module  string // 模块名
	Handler string // 函数名，如 app.overviewProbe

	// 请求信息
	JSONBody    *SchemaInfo // ShouldBindJSON 类型
	QueryType   *SchemaInfo // ShouldBindQuery 类型
	FormData    bool        // multipart/form-data
	PathParams  []ParamDef  // c.Param 参数
	QueryParams []ParamDef  // c.Query / c.DefaultQuery 参数
	IsWS        bool        // WebSocket 路由
	IsSSE       bool        // SSE 路由

	// 响应信息
	ResponseBody     *SchemaInfo // respondSuccess 第三个参数的类型（结构体）
	ResponseBodyType string      // 响应类型的 Go 类型字符串（用于 slice/map 等复杂类型）
}

type ParamDef struct {
	Name        string
	Type        string // string, integer, boolean
	Required    bool
	Description string
}

// ─── 结构体字段中间表示 ──────────────────────────────

type SchemaInfo struct {
	PkgName  string
	TypeName string
	Fields   []FieldInfo
}

type FieldInfo struct {
	Name        string // json tag name
	Type        string // go type as string
	Required    bool   // has binding:"required"
	Description string // from description tag or field comment
}

// ─── 项目配置 ────────────────────────────────────────

// ProjectConfig 定义项目的目录结构和 OpenAPI 元数据
// 修改 DefaultConfig() 即可适配不同项目
type ProjectConfig struct {
	// 目录配置（相对于 projectRoot）
	CtrlDir        string   // handler 文件目录
	CtrlPrefix     string   // handler 文件名前缀
	ServiceDir     string   // service 层目录
	TypeDirs       []string // 额外的类型定义目录
	ResponseFile   string   // 响应结构体定义文件
	ResponseStruct string   // 响应结构体名称

	// OpenAPI 元数据
	Title      string // API 标题
	Version    string // API 版本
	ServerURL  string // 服务 URL 前缀
	ServerDesc string // 服务描述

	// 输出
	OutputFile string // 输出文件路径
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ProjectConfig {
	return &ProjectConfig{
		CtrlDir:        "internal/server",
		CtrlPrefix:     "ctrl_",
		ServiceDir:     "internal/service",
		TypeDirs:       []string{"config", "pkgs"},
		ResponseFile:   "internal/server/response.go",
		ResponseStruct: "APIResponse",
		Title:          "isrvd API",
		Version:        "1.0.0",
		ServerURL:      "/api",
		ServerDesc:     "API 服务",
		OutputFile:     "public/openapi/apis.json",
	}
}

// ─── 全局状态 ────────────────────────────────────────

var (
	projectRoot string
	cfg         *ProjectConfig
	structCache = map[string]*SchemaInfo{} // "pkg.TypeName" -> schema
	routes      []RouteDef
	fileCache   = map[string]*ast.File{} // 缓存已解析的文件
	fsetCache   = token.NewFileSet()     // 复用 FileSet
)

// 整数类型集合（用于 buildProperty 和 extractDefaultQueryParam 的快速判断）
var intTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
}

// 标准库类型，避免在 buildProperty 中生成无效 $ref
var stdTypes = map[string]bool{
	"time.Time":     true,
	"time.Duration": true,
}

// ─── 入口 ────────────────────────────────────────────
var outFile = flag.String("o", "", "输出文件路径（覆盖默认值）")

func main() {
	flag.Parse()
	var err error
	projectRoot, err = os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化配置
	cfg = DefaultConfig()
	if *outFile != "" {
		cfg.OutputFile = *outFile
	}

	// 0. 预注册 APIResponse schema（从源文件提取字段注释）
	responseFile := filepath.Join(projectRoot, cfg.ResponseFile)
	if schema := parseServiceStruct("", cfg.ResponseStruct, responseFile); schema != nil {
		structCache[cfg.ResponseStruct] = schema
	} else {
		// 降级处理：如果无法从文件提取，使用默认值
		structCache[cfg.ResponseStruct] = &SchemaInfo{
			TypeName: cfg.ResponseStruct,
			Fields: []FieldInfo{
				{Name: "success", Type: "bool", Required: true, Description: "请求是否成功"},
				{Name: "message", Type: "string", Description: "提示信息"},
				{Name: "payload", Type: "any", Description: "响应数据负载"},
			},
		}
	}

	// 1-2. 解析 ctrl 文件 & 预收集类型信息
	ctrlDir := filepath.Join(projectRoot, cfg.CtrlDir)
	entries, err := os.ReadDir(ctrlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取目录失败: %v\n", err)
		os.Exit(1)
	}

	ctrlFiles := getCtrlFiles(ctrlDir, entries)
	for _, filename := range ctrlFiles {
		parseCtrlFile(filename)
		collectAllTypesFromFile(filename)
	}

	// 2.5 解析 service 及额外类型目录中的类型定义
	collectServiceTypes()

	// 3. 对每个路由的 handler 分析请求类型
	for i := range routes {
		analyzeHandler(&routes[i])
	}

	// 4. 生成 OpenAPI JSON
	openapi := buildOpenAPI()
	out, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化失败: %v\n", err)
		os.Exit(1)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(cfg.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(cfg.OutputFile, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "OpenAPI 文档已生成: %s (%d bytes)\n", cfg.OutputFile, len(out))
}



// ─── 第 1 步：解析 Route 定义 ─────────────────────────

// parseFile 解析单个文件，结果缓存
func parseFile(filename string) *ast.File {
	if cached, ok := fileCache[filename]; ok {
		return cached
	}
	f, err := parser.ParseFile(fsetCache, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	fileCache[filename] = f
	return f
}

// getCtrlFiles 从目录 entries 中筛选 handler 文件并返回完整路径列表
func getCtrlFiles(dir string, entries []os.DirEntry) []string {
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), cfg.CtrlPrefix) && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}

func parseCtrlFile(filename string) {
	f := parseFile(filename)
	if f == nil {
		fmt.Fprintf(os.Stderr, "解析 %s 失败\n", filename)
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			return true
		}

		// 匹配 define*Routes() 方法
		if !strings.HasPrefix(funcDecl.Name.Name, "define") || !strings.HasSuffix(funcDecl.Name.Name, "Routes") {
			return true
		}

		// 在函数体中查找返回语句中的 []Route 复合字面量
		ast.Inspect(funcDecl.Body, func(n2 ast.Node) bool {
			retStmt, ok := n2.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, expr := range retStmt.Results {
				composite, ok := expr.(*ast.CompositeLit)
				if !ok {
					continue
				}
				// 检查是否是 []Route
				arrType, ok := composite.Type.(*ast.ArrayType)
				if !ok {
					continue
				}
				sel, ok := arrType.Elt.(*ast.Ident)
				if !ok || sel.Name != "Route" {
					continue
				}

				// 解析数组中的每个 Route 字面量
				for _, elt := range composite.Elts {
					cl, ok := elt.(*ast.CompositeLit)
					if !ok {
						continue
					}
					route := parseRouteLiteral(cl)
					if route.Method != "" {
						routes = append(routes, route)
					}
				}
			}
			return true
		})
		return true
	})
}

func parseRouteLiteral(cl *ast.CompositeLit) RouteDef {
	var r RouteDef
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Method":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Method = strings.Trim(lit.Value, `"`)
			}
		case "Path":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Path = strings.Trim(lit.Value, `"`)
			}
		case "Label":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Label = strings.Trim(lit.Value, `"`)
			}
		case "Module":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				r.Module = strings.Trim(lit.Value, `"`)
			}
		case "Handler":
			if sel, ok := kv.Value.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					r.Handler = x.Name + "." + sel.Sel.Name
				}
			}
		}
	}
	return r
}

// ─── 第 2a 步：预扫描所有类型引用 ─────────────────────

// findFuncReturnType 在指定文件中查找函数的返回类型名称
// collectPathParamsFromFunc 分析指定函数的 body，提取其中的 c.Param 调用
func collectPathParamsFromFunc(filename, funcName string, r *RouteDef) {
	f := parseFile(filename)
	if f == nil {
		return
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName || fd.Body == nil {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Param" {
				return true
			}
			if len(call.Args) >= 1 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					name := strings.Trim(lit.Value, `"`)
					if !hasParam(r.PathParams, name) {
						r.PathParams = append(r.PathParams, ParamDef{
							Name: name, Type: "string", Required: true,
						})
					}
				}
			}
			return true
		})
	}
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

func findFuncReturnType(filename, funcName string) string {
	f := parseFile(filename)
	if f == nil {
		return ""
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName || fd.Type.Results == nil {
			continue
		}
		// 返回类型可能是 (Type, bool) 形式的 tuple
		for _, field := range fd.Type.Results.List {
			if sel, ok := field.Type.(*ast.SelectorExpr); ok {
				return fullSelName(sel)
			}
			if ident, ok := field.Type.(*ast.Ident); ok {
				if ident.Name != "bool" && ident.Name != "error" {
					return ident.Name
				}
			}
		}
	}
	return ""
}

// collectServiceTypes 遍历 service 及额外类型目录，解析所有类型定义
func collectServiceTypes() {
	// 遍历 service 目录
	serviceDir := filepath.Join(projectRoot, cfg.ServiceDir)
	collectTypesFromDir(serviceDir)

	// 遍历额外的类型定义目录
	for _, dir := range cfg.TypeDirs {
		collectTypesFromDir(filepath.Join(projectRoot, dir))
	}
}

// collectTypesFromDir 遍历目录，解析所有 Go 文件中的类型定义
func collectTypesFromDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			// 递归遍历子目录
			subDir := filepath.Join(dir, entry.Name())
			collectTypesFromDir(subDir)
			continue
		}
		
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		
		filename := filepath.Join(dir, entry.Name())
		collectTypeDefinitionsFromFile(filename)
	}
}

// collectTypeDefinitionsFromFile 解析文件中的顶层类型定义
func collectTypeDefinitionsFromFile(filename string) {
	f := parseFile(filename)
	if f == nil {
		return
	}
	
	// 从文件路径提取包名
	pkgName := extractPkgNameFromPath(filename)
	
	for _, decl := range f.Decls {
		// 查找类型定义: type Xxx struct { ... }
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			
			typeName := ts.Name.Name
			
			// 构建完整的类型名 (包名.类型名)
			var fullTypeName string
			if pkgName != "" {
				fullTypeName = pkgName + "." + typeName
			}
			
			if fullTypeName != "" {
				resolveStructSchema(fullTypeName, nil, filename)
			}
		}
	}
}

// extractPkgNameFromPath 从文件路径提取包名
// 根据配置的目录结构动态推断包名
func extractPkgNameFromPath(filename string) string {
	// 处理 service 目录（如 internal/service/account/member.go → account）
	svcMarker := "/" + filepath.ToSlash(cfg.ServiceDir) + "/"
	if idx := strings.Index(filename, svcMarker); idx != -1 {
		rest := filename[idx+len(svcMarker):]
		subDir := strings.SplitN(rest, "/", 2)[0]
		return subDir
	}

	// 处理额外的类型目录
	for _, dir := range cfg.TypeDirs {
		marker := "/" + filepath.ToSlash(dir) + "/"
		if idx := strings.Index(filename, marker); idx != -1 {
			rest := filename[idx+len(marker):]
			// 如果 rest 中还有子目录，取第一层作为包名
			if subDir := strings.SplitN(rest, "/", 2)[0]; !strings.HasSuffix(subDir, ".go") {
				return subDir
			}
			// 否则直接用目录名作为包名（如 config/types.go → config）
			return filepath.Base(dir)
		}
	}

	return ""
}

func collectAllTypesFromFile(filename string) {
	f := parseFile(filename)
	if f == nil {
		return
	}

	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		// 收集所有函数体中的类型引用
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "ShouldBindJSON" {
				return true
			}
			// 在函数体中反向查找 var req Xxx 声明 或 req := &Xxx{} 赋值
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				// 模式1: var req Xxx — DeclStmt
				if ds, ok := n2.(*ast.DeclStmt); ok {
					gd, ok := ds.Decl.(*ast.GenDecl)
					if !ok || gd.Tok != token.VAR {
						return true
					}
					for _, spec := range gd.Specs {
						if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) == 1 {
							typeName := typeExprToString(vs.Type)
							if sel, ok := vs.Type.(*ast.SelectorExpr); ok {
								typeName = fullSelName(sel)
							}
							if typeName != "" && typeName != "struct" {
								resolveStructSchema(typeName, nil, filename)
							}
						}
					}
					return true
				}
				// 模式2: req := &pkg.Type{} — AssignStmt
				if as, ok := n2.(*ast.AssignStmt); ok && as.Tok == token.DEFINE && len(as.Rhs) == 1 {
					if comp, ok := as.Rhs[0].(*ast.CompositeLit); ok {
						typeName := typeExprToString(comp.Type)
						if typeName != "" && typeName != "struct" {
							resolveStructSchema(typeName, nil, filename)
						}
					}
				}
				return true
			})
			return true
		})
	}
}

// ─── 第 3 步：分析 handler 函数体 ─────────────────────

func analyzeHandler(r *RouteDef) {
	// 找到 handler 对应的 ctrl_*.go 文件
	handlerParts := strings.SplitN(r.Handler, ".", 2)
	if len(handlerParts) != 2 {
		return
	}
	funcName := handlerParts[1]

	ctrlDir := filepath.Join(projectRoot, cfg.CtrlDir)
	entries, err := os.ReadDir(ctrlDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), cfg.CtrlPrefix) || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filename := filepath.Join(ctrlDir, entry.Name())
		f := parseFile(filename)
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
			localTypes := collectLocalTypes(f, fsetCache, funcName)

			// 分析函数体
			analyzeFuncBody(r, fd.Body, fsetCache, localTypes, filename)
			return
		}
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

func collectLocalTypes(f *ast.File, fset *token.FileSet, _ string) map[string]string {
	result := make(map[string]string)
	typeAliases := make(map[string]string) // 本地别名 → 原始类型

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			typeName := ts.Name.Name
			if ts.Assign.IsValid() {
				// type X = Y 类型别名
				collectTypeAlias(ts, typeAliases)
			} else if st, ok := ts.Type.(*ast.StructType); ok {
				result[typeName] = structTypeToJSONSchema(st, fset)
			}
		}
	}

	// 将别名信息也保存（用特殊前缀标记）
	for alias, target := range typeAliases {
		result["__alias__"+alias] = target
	}
	return result
}

// collectTypeAlias 从 TypeSpec 中收集类型别名
func collectTypeAlias(ts *ast.TypeSpec, typeAliases map[string]string) {
	typeName := ts.Name.Name
	if sel, ok := ts.Type.(*ast.SelectorExpr); ok {
		typeAliases[typeName] = fullSelName(sel)
	} else if ident, ok := ts.Type.(*ast.Ident); ok {
		typeAliases[typeName] = ident.Name
	}
}

func inlineStructToSchema(st *ast.StructType) *SchemaInfo {
	schema := &SchemaInfo{TypeName: "InlineRequest"}
	if st.Fields == nil {
		return schema
	}
	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue
		}
		for _, name := range field.Names {
			fieldInfo := extractFieldInfo(name.Name, field)
			if fieldInfo.Name != "-" {
				schema.Fields = append(schema.Fields, fieldInfo)
			}
		}
	}
	return schema
}

func structTypeToJSONSchema(st *ast.StructType, _ *token.FileSet) string {
	var fields []string
	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue
		}
		fieldInfo := extractFieldInfo(field.Names[0].Name, field)
		if fieldInfo.Name != "-" {
			goType := typeExprToString(field.Type)
			fields = append(fields, fmt.Sprintf("%s:%s:%v", fieldInfo.Name, goType, fieldInfo.Required))
		}
	}
	return strings.Join(fields, ";")
}

// handlerAnalysisState 保存 handler 函数分析过程中的状态
type handlerAnalysisState struct {
	varTypes     map[string]string  // 变量名 → 类型名（如 "resp" → "account.LoginResponse"）
	localTypes   map[string]string  // 本地类型别名
	filename     string            // 当前分析的文件
	responseBody *SchemaInfo       // 提取到的响应类型
}

func analyzeFuncBody(r *RouteDef, body *ast.BlockStmt, fset *token.FileSet, localTypes map[string]string, filename string) {
	state := &handlerAnalysisState{
		varTypes:   make(map[string]string),
		localTypes: localTypes,
		filename:   filename,
	}

	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			analyzeAssignStmtV2(stmt, r, state)
		case *ast.DeclStmt:
			analyzeDeclStmtV2(stmt, r, state)
		case *ast.CallExpr:
			analyzeCallExprV2(stmt, r, state)
		}
		return true
	})

	r.ResponseBody = state.responseBody
}

// analyzeAssignStmtV2 处理赋值语句，跟踪变量类型
func analyzeAssignStmtV2(stmt *ast.AssignStmt, r *RouteDef, state *handlerAnalysisState) {
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

		// 情况 1: resp := &pkg.Type{} 或 resp := pkg.Type{}
		if comp, ok := stmt.Rhs[0].(*ast.CompositeLit); ok {
			typeName := typeExprToString(comp.Type)
			if typeName != "" && typeName != "struct" && len(lhsNames) > 0 {
				state.varTypes[lhsNames[0]] = typeName
			}
			return
		}

		// 情况 2: resp, err := app.svc.Method()
		if callExpr, ok := stmt.Rhs[0].(*ast.CallExpr); ok && len(lhsNames) >= 1 {
			retType := resolveCallReturnType(callExpr, state.filename)
			if retType != "" {
				state.varTypes[lhsNames[0]] = retType
			}
			return
		}
	}

	// 处理普通赋值 =
	// 如: result = app.svc.Method()
	if len(stmt.Lhs) == 1 && len(stmt.Rhs) == 1 {
		if ident, ok := stmt.Lhs[0].(*ast.Ident); ok {
			if callExpr, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
				retType := resolveCallReturnType(callExpr, state.filename)
				if retType != "" {
					state.varTypes[ident.Name] = retType
				}
			}
		}
	}
}

// analyzeDeclStmtV2 处理声明语句，跟踪变量类型
func analyzeDeclStmtV2(stmt *ast.DeclStmt, r *RouteDef, state *handlerAnalysisState) {
	gd, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return
	}

	for _, spec := range gd.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) == 1 {
			varName := vs.Names[0].Name
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
					structCache[cacheKey] = schema
					state.varTypes[varName] = cacheKey
					resolveNestedTypesInSchema(schema, "")
				}
			}
		}
	}
}

// analyzeCallExprV2 处理函数调用表达式
func analyzeCallExprV2(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	// 处理直接函数调用（如 respondSuccess、respondError）
	if ident, ok := stmt.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "respondSuccess":
			if len(stmt.Args) >= 3 {
				analyzeRespondSuccessV2(stmt, r, state)
			}
			return
		case "respondError":
			// 错误处理，无需特殊处理
			return
		default:
			collectPathParamsFromFunc(state.filename, ident.Name, r)
			return
		}
	}

	sel, ok := stmt.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	switch sel.Sel.Name {
	case "ShouldBindJSON":
		analyzeShouldBindJSON(stmt, r, state)
	case "ShouldBindQuery":
		analyzeShouldBindQuery(stmt, r, state)
	case "Query":
		extractQueryParam(stmt, r, "string", false)
	case "DefaultQuery":
		extractDefaultQueryParam(stmt, r)
	case "Param":
		extractParamCall(stmt, r)
	case "PostForm":
		r.FormData = true
		extractQueryParam(stmt, r, "string", false)
	case "FormFile":
		r.FormData = true
	case "ParseMultipartForm":
		r.FormData = true
	case "NewEventWriter":
		r.IsSSE = true
	}

	// 检测 WebSocket
	if sel2, ok2 := sel.X.(*ast.SelectorExpr); ok2 && sel2.Sel.Name == "wsConfig" {
		r.IsWS = true
	}
}

// analyzeShouldBindJSON 分析 ShouldBindJSON 调用
func analyzeShouldBindJSON(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
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
		schema := resolveStructSchema(typeName, state.localTypes, state.filename)
		if schema != nil {
			r.JSONBody = schema
		}
	}
}

// analyzeShouldBindQuery 分析 ShouldBindQuery 调用
func analyzeShouldBindQuery(stmt *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
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
		schema := resolveStructSchema(typeName, state.localTypes, state.filename)
		if schema != nil {
			r.QueryType = schema
		}
	}
}

// analyzeRespondSuccessV2 分析 respondSuccess(c, msg, data) 调用
func analyzeRespondSuccessV2(call *ast.CallExpr, r *RouteDef, state *handlerAnalysisState) {
	if len(call.Args) < 3 {
		return
	}

	dataArg := call.Args[2]

	// 情况 1: nil → 无 payload
	if ident, ok := dataArg.(*ast.Ident); ok && ident.Name == "nil" {
		return
	}

	// 情况 2: 变量引用，如 resp → 查找变量类型
	if ident, ok := dataArg.(*ast.Ident); ok {
		typeName, ok := state.varTypes[ident.Name]
		if ok && typeName != "" {
			r.ResponseBodyType = typeName
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
		retType := resolveCallReturnType(callExpr, state.filename)
		if retType != "" {
			r.ResponseBodyType = retType
		}
		return
	}
}

// resolveCallReturnType 解析函数调用的返回类型
func resolveCallReturnType(call *ast.CallExpr, filename string) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	methodName := sel.Sel.Name

	// 处理链式调用: app.accountSvc.Login
	if recvSel, ok := sel.X.(*ast.SelectorExpr); ok {
		svcVarName := recvSel.Sel.Name
		pkgName := inferPkgFromVar(filename, svcVarName)
		if pkgName != "" {
			// findMethodReturnType 已经处理了 import 别名解析和 pkgName 前缀
			return findMethodReturnType(filename, pkgName, methodName)
		}
	}

	return ""
}

// resolveImportAlias 解析导入别名，将别名.类型名 转换为实际的包名.类型名
// 例如: pkgApisix.Consumer → apisix.Consumer
func resolveImportAlias(filename, typeName string) string {
	// 检查是否包含 "."，且第一部分可能是导入别名
	idx := strings.Index(typeName, ".")
	if idx == -1 {
		return typeName
	}
	
	alias := typeName[:idx]
	typeSuffix := typeName[idx:] // 包含 "."
	
	// 从文件中提取导入别名映射
	importMap := extractImportAliases(filename)
	
	// 如果别名在映射中，替换为实际的包名
	if realPkg, ok := importMap[alias]; ok {
		return realPkg + typeSuffix
	}
	
	return typeName
}

// extractImportAliases 从 Go 文件中提取导入别名映射
// 返回 map[别名]实际包名
func extractImportAliases(filename string) map[string]string {
	result := make(map[string]string)
	
	f := parseFile(filename)
	if f == nil {
		return result
	}
	
	for _, imp := range f.Imports {
		// 导入路径，如 "isrvd/pkgs/apisix"
		importPath := strings.Trim(imp.Path.Value, "\"")
		
		// 提取包名（路径最后一段）
		parts := strings.Split(importPath, "/")
		realPkgName := parts[len(parts)-1]
		
		if imp.Name != nil {
			// 有别名: pkgApisix "isrvd/pkgs/apisix"
			alias := imp.Name.Name
			result[alias] = realPkgName
		} else {
			// 无别名: "isrvd/pkgs/apisix"
			result[realPkgName] = realPkgName
		}
	}
	
	return result
}

// inferPkgFromVar 从变量名推断包名
func inferPkgFromVar(filename, varName string) string {
	if strings.HasSuffix(varName, "Svc") {
		return strings.ToLower(strings.TrimSuffix(varName, "Svc"))
	}
	return ""
}

// findMethodReturnType 在 service 包中查找方法的返回类型
func findMethodReturnType(ctrlFile, pkgName, methodName string) string {
	searchDir := filepath.Join(projectRoot, cfg.ServiceDir, pkgName)
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filename := filepath.Join(searchDir, entry.Name())
		retType := findMethodReturnTypeInFile(filename, methodName)
		if retType != "" {
			// 解析返回类型中的 import 别名
			retType = resolveReturnType(filename, pkgName, retType)
			return retType
		}
	}
	return ""
}

// resolveReturnType 将方法返回类型解析为最终的 schema 名称格式
// 处理 slice、指针、import 别名等情况
// 输入如: []pkgApisix.Consumer, *LoginResponse, []*config.PasskeyCredential
// 输出如: apisix.[]Consumer, account.LoginResponse, config.[]PasskeyCredential
func resolveReturnType(svcFilename, pkgName, retType string) string {
	// 提取 slice 前缀
	slicePrefix := ""
	inner := retType
	if strings.HasPrefix(inner, "[]") {
		slicePrefix = "[]"
		inner = inner[2:]
	}
	// 去指针
	inner = strings.TrimPrefix(inner, "*")
	
	// 解析 import 别名
	if strings.Contains(inner, ".") {
		inner = resolveImportAlias(svcFilename, inner)
	} else {
		// 不包含 "."，说明是本包类型，添加 pkgName 前缀
		inner = pkgName + "." + inner
	}
	
	// 重新组装: pkgName.[]TypeName 格式
	if slicePrefix != "" {
		// 将 "pkg.TypeName" → "pkg.[]TypeName"
		if dotIdx := strings.LastIndex(inner, "."); dotIdx != -1 {
			return inner[:dotIdx+1] + slicePrefix + inner[dotIdx+1:]
		}
		return slicePrefix + inner
	}
	
	return inner
}

// findMethodReturnTypeInFile 在单个文件中查找方法的返回类型
func findMethodReturnTypeInFile(filename, methodName string) string {
	f := parseFile(filename)
	if f == nil {
		return ""
	}

	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != methodName || fd.Type.Results == nil {
			continue
		}
		for _, field := range fd.Type.Results.List {
			goType := typeExprToString(field.Type)
			if goType == "error" {
				continue
			}
			if goType == "bool" {
				continue
			}
			// 过滤掉 any、interface{} 等空接口类型
			if goType == "any" || goType == "interface{}" {
				continue
			}
			// 去除所有指针前缀
			goType = strings.TrimPrefix(goType, "*")
			for strings.HasPrefix(goType, "*") {
				goType = goType[1:]
			}
			return goType
		}
	}
	return ""
}

// extractHandlerName 从 Handler 字符串提取函数名
func extractHandlerName(handler string) string {
	parts := strings.SplitN(handler, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
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

func fullSelName(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name + "." + sel.Sel.Name
	}
	return sel.Sel.Name
}

func typeExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fullSelName(t)
	case *ast.StarExpr:
		return "*" + typeExprToString(t.X)
	case *ast.ArrayType:
		return "[]" + typeExprToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprToString(t.Key) + "]" + typeExprToString(t.Value)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// ─── 类型解析 ─────────────────────────────────────────

func resolveStructSchema(typeName string, localTypes map[string]string, ctrlFile string) *SchemaInfo {
	// 1. 检查本地类型别名（type X = pkg.Type）
	if localTypes != nil {
		if aliasTarget, ok := localTypes["__alias__"+typeName]; ok {
			typeName = aliasTarget
		}
	}

	// 2. 检查本地 struct 类型
	if localTypes != nil {
		for name, fields := range localTypes {
			if strings.HasPrefix(name, "__alias__") {
				continue
			}
			if name == typeName || strings.HasPrefix(typeName, name+".") {
				info := parseFieldsString(name, fields)
				if info != nil {
					// 缓存本地类型到 structCache，避免孤立引用
					cacheKey := name
					structCache[cacheKey] = info
					// 递归解析嵌套结构体字段类型
					resolveNestedTypesInSchema(info, ctrlFile)
					return info
				}
			}
		}
	}

	// 2. 检查是否为带包名的类型
	parts := strings.SplitN(typeName, ".", 2)
	var pkgAlias, structName string
	if len(parts) == 2 {
		pkgAlias = parts[0]
		structName = parts[1]
	} else {
		structName = typeName
	}

	// 3. 从缓存中查找
	cacheKey := pkgAlias + "." + structName
	if cached, ok := structCache[cacheKey]; ok {
		return cached
	}

	// 4. 解析对应的 service 文件
	schema := parseServiceStruct(pkgAlias, structName, ctrlFile)
	if schema != nil {
		// 使用规范化名称作为缓存 key
		normalizedKey := normalizeSchemaName(cacheKey)
		structCache[normalizedKey] = schema
		// 同时也用原始 key
		structCache[cacheKey] = schema

		// 5. 递归解析嵌套结构体字段类型（如 *config.AgentConfig）
		resolveNestedTypesInSchema(schema, ctrlFile)
	}
	return schema
}

// resolveNestedTypesInSchema 解析 schema 字段中引用的嵌套结构体类型
func resolveNestedTypesInSchema(schema *SchemaInfo, ctrlFile string) {
	for _, field := range schema.Fields {
		goType := strings.TrimPrefix(field.Type, "*")
		goType = strings.TrimPrefix(goType, "[]")
		goType = strings.TrimPrefix(goType, "*")
		if !strings.Contains(goType, ".") || intTypes[goType] {
			continue
		}
		if goType == "json.RawMessage" || goType == "any" || goType == "interface{}" {
			continue
		}
		resolveStructSchema(goType, nil, ctrlFile)
	}
}

// normalizeAliasToDir 将 import 别名规范化为包目录名
// 规则：删除 svc/pkg/lib 前缀，然后转小写
func normalizeAliasToDir(alias string) string {
	if alias == "" {
		return ""
	}
	// 删除 svc/pkg/lib 前缀
	if strings.HasPrefix(alias, "svc") && len(alias) > 3 {
		alias = alias[3:]
	} else if strings.HasPrefix(alias, "pkg") && len(alias) > 3 {
		alias = alias[3:]
	} else if strings.HasPrefix(alias, "lib") && len(alias) > 3 {
		alias = alias[3:]
	}
	return strings.ToLower(alias)
}

func parseServiceStruct(pkgAlias, structName string, ctrlFile string) *SchemaInfo {
	serviceDir := filepath.Join(projectRoot, cfg.ServiceDir)

	// 构建搜索目录列表
	searchDirs := buildSearchDirs(serviceDir, pkgAlias)

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			filename := filepath.Join(dir, entry.Name())
			schema := findStructInFile(filename, structName, pkgAlias)
			if schema != nil {
				return schema
			}
		}
	}
	return nil
}

// buildSearchDirs 构建需要搜索的目录列表
func buildSearchDirs(serviceDir, pkgAlias string) []string {
	var dirs []string
	if pkgAlias != "" {
		dirs = append(dirs, filepath.Join(serviceDir, pkgAlias))
		// 额外类型目录
		for _, td := range cfg.TypeDirs {
			dirs = append(dirs, filepath.Join(projectRoot, td, pkgAlias))
		}
		dirs = append(dirs, filepath.Join(projectRoot, pkgAlias)) // 根级包
	}

	if normalized := normalizeAliasToDir(pkgAlias); normalized != "" && normalized != pkgAlias {
		dirs = append(dirs, filepath.Join(serviceDir, normalized))
		for _, td := range cfg.TypeDirs {
			dirs = append(dirs, filepath.Join(projectRoot, td, normalized))
		}
		dirs = append(dirs, filepath.Join(projectRoot, normalized)) // 根级包
	}
	return dirs
}

// findStructInFile 在单个文件中查找结构体定义
func findStructInFile(filename, structName, pkgAlias string) *SchemaInfo {
	f := parseFile(filename)
	if f == nil {
		return nil
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			// 提取结构体注释（GenDecl.Doc 或 TypeSpec.Doc）
			structComment := extractComment(gd.Doc)
			if structComment == "" {
				structComment = extractComment(ts.Doc)
			}
			return buildSchemaFromStruct(pkgAlias, structName, st, filename, structComment)
		}
	}
	return nil
}

// extractComment 从 AST CommentGroup 中提取注释文本
func extractComment(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var lines []string
	for _, c := range cg.List {
		line := strings.TrimPrefix(c.Text, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		lines = append(lines, strings.TrimSpace(line))
	}
	return strings.Join(lines, " ")
}

// buildSchemaFromStruct 从 AST StructType 构建 SchemaInfo
func buildSchemaFromStruct(pkgAlias, structName string, st *ast.StructType, filename, structComment string) *SchemaInfo {
	schema := &SchemaInfo{
		PkgName:  pkgAlias,
		TypeName: structName,
	}

	// 读取文件内容以获取字段注释
	fieldComments := make(map[string]string)
	if filename != "" {
		fieldComments = extractFieldComments(filename, st)
	}

	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue // embedded
		}
		for _, name := range field.Names {
			fieldInfo := extractFieldInfo(name.Name, field)

			// 优先使用字段注释，其次使用 json tag 中的 description
			if comment, ok := fieldComments[name.Name]; ok && comment != "" {
				fieldInfo.Description = comment
			}

			if fieldInfo.Name != "-" {
				schema.Fields = append(schema.Fields, fieldInfo)
			}
		}
	}
	return schema
}

// extractFieldComments 从结构体定义中提取字段注释
// 支持三种格式：
//  1. 字段上方注释：// 字段描述
//  2. 字段后面注释：FieldName Type `json:"..."` // 字段描述
//  3. tag 中的 description："description" 字段描述
func extractFieldComments(filename string, st *ast.StructType) map[string]string {
	comments := make(map[string]string)

	// 读取文件内容
	content, err := os.ReadFile(filename)
	if err != nil {
		return comments
	}
	lines := strings.Split(string(content), "\n")

	// 遍历结构体字段，查找注释
	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue
		}

		// 获取字段所在的行号（AST 行号从 1 开始）
		fieldLine := fsetCache.Position(field.Pos()).Line

		// 1. 检查字段所在行的行内注释（trailing comment）
		if fieldLine <= len(lines) {
			lineText := lines[fieldLine-1] // 数组索引从 0 开始
			if idx := strings.Index(lineText, "//"); idx >= 0 {
				commentText := strings.TrimSpace(lineText[idx+2:])
				// 排除 tag 定义部分，只取注释
				for _, name := range field.Names {
					comments[name.Name] = commentText
				}
			}
		}

		// 2. 向上查找字段上方注释
		if fieldLine > 1 {
			commentLine := fieldLine - 1
			if commentLine <= len(lines) {
				prevLine := strings.TrimSpace(lines[commentLine-1])
				// 如果上一行是注释，且当前行不是注释（避免重复）
				if strings.HasPrefix(prevLine, "//") && !strings.HasPrefix(strings.TrimSpace(lines[fieldLine-1]), "//") {
					commentText := strings.TrimSpace(strings.TrimPrefix(prevLine, "//"))

					// 检查注释格式：
					// 1. "FieldName 描述" - 明确指定字段名
					// 2. "描述" - 直接是描述
					for _, name := range field.Names {
						if strings.HasPrefix(commentText, name.Name+" ") {
							// 格式："FieldName 描述"
							comments[name.Name] = strings.TrimPrefix(commentText, name.Name+" ")
						} else if comments[name.Name] == "" {
							// 格式："描述"（假设短注释是描述，且没有被行内注释设置过）
							comments[name.Name] = commentText
						}
					}
				}
			}
		}
	}

	return comments
}

// extractFieldInfo 从结构体字段中提取信息
func extractFieldInfo(fieldName string, field *ast.Field) FieldInfo {
	goType := typeExprToString(field.Type)
	jsonName := fieldName
	required := false
	description := ""

	if field.Tag == nil {
		return FieldInfo{Name: jsonName, Type: goType, Required: required, Description: description}
	}

	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))

	// 优先 json tag，其次 form tag（都是取逗号前的部分）
	if jt := tag.Get("json"); jt != "" {
		jsonName, _, _ = strings.Cut(jt, ",")
	} else if ft := tag.Get("form"); ft != "" {
		jsonName, _, _ = strings.Cut(ft, ",")
	}

	// 检查 required 标记
	if binding := tag.Get("binding"); binding != "" && strings.Contains(binding, "required") {
		required = true
	}

	// 提取 description tag 作为字段描述
	if desc := tag.Get("description"); desc != "" {
		description = desc
	}

	return FieldInfo{Name: jsonName, Type: goType, Required: required, Description: description}
}

func parseFieldsString(name string, s string) *SchemaInfo {
	info := &SchemaInfo{TypeName: name}
	for _, field := range strings.Split(s, ";") {
		if field == "" {
			continue
		}
		parts := strings.SplitN(field, ":", 3)
		if len(parts) >= 2 {
			required := len(parts) >= 3 && parts[2] == "true"
			info.Fields = append(info.Fields, FieldInfo{
				Name:     parts[0],
				Type:     parts[1],
				Required: required,
			})
		}
	}
	return info
}

// ─── 构建 OpenAPI ─────────────────────────────────────

func buildOpenAPI() *OpenAPI {
	oa := &OpenAPI{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:   cfg.Title,
			Version: cfg.Version,
		},
		Servers: []OpenAPIServer{
			{URL: cfg.ServerURL, Description: cfg.ServerDesc},
		},
		Paths: make(map[string]map[string]any),
		Components: OpenAPIComponents{
			Schemas: make(map[string]any),
			SecuritySchemes: map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}

	// 收集所有 schema
	allSchemas := collectAllSchemas()
	for name, schema := range allSchemas {
		schemaDef := buildSchemaObject(schema, allSchemas)
		oa.Components.Schemas[name] = schemaDef
	}

	// 按模块分组路由
	modules := make(map[string][]RouteDef)
	for _, r := range routes {
		modules[r.Module] = append(modules[r.Module], r)
	}

	// 排序模块名
	var moduleNames []string
	for m := range modules {
		moduleNames = append(moduleNames, m)
	}
	sort.Strings(moduleNames)

	for _, mod := range moduleNames {
		for _, r := range modules[mod] {
			addPathItem(oa, r, allSchemas)
		}
	}

	return oa
}

func collectAllSchemas() map[string]*SchemaInfo {
	result := make(map[string]*SchemaInfo)
	for cacheKey, schema := range structCache {
		// 只保留规范化名称
		normalized := normalizeSchemaName(cacheKey)
		result[normalized] = schema
	}
	return result
}

func buildSchemaObject(schema *SchemaInfo, allSchemas map[string]*SchemaInfo) map[string]any {
	props := make(map[string]any)
	var required []string

	for _, f := range schema.Fields {
		prop := buildProperty(f.Type, allSchemas)

		// 添加字段描述（如果有）
		if f.Description != "" {
			prop["description"] = f.Description
		} else {
			// 为缺少注释的字段使用字段名作为默认描述
			prop["description"] = f.Name
		}

		props[f.Name] = prop
		if f.Required {
			required = append(required, f.Name)
		}
	}

	result := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func buildProperty(goType string, allSchemas map[string]*SchemaInfo) map[string]any {
	// 处理指针类型
	goType = strings.TrimPrefix(goType, "*")

	switch {
	case goType == "string":
		return map[string]any{"type": "string"}
	case intTypes[goType]:
		return map[string]any{"type": "integer"}
	case goType == "float64" || goType == "float32":
		return map[string]any{"type": "number"}
	case goType == "bool":
		return map[string]any{"type": "boolean"}
	case strings.HasPrefix(goType, "[]"):
		itemType := strings.TrimPrefix(goType, "[]")
		itemType = strings.TrimPrefix(itemType, "*")
		items := buildProperty(itemType, allSchemas)
		return map[string]any{"type": "array", "items": items}
	case strings.HasPrefix(goType, "map["):
		return map[string]any{"type": "object"}
	case goType == "json.RawMessage":
		return map[string]any{"type": "object"}
	case goType == "any" || goType == "interface{}":
		return map[string]any{} // any type
	default:
		// 标准库类型降级为基本类型
		if stdTypes[goType] {
			return map[string]any{"type": "string"}
		}
		// 尝试引用其他 schema
		parts := strings.SplitN(goType, ".", 2)
		if len(parts) == 2 {
			refName := parts[0] + "." + parts[1]
			return map[string]any{"$ref": "#/components/schemas/" + sanitizeRef(refName)}
		}
		// 未知类型默认为 object
		return map[string]any{"type": "object"}
	}
}

// normalizeSchemaName 将 Go import 别名映射为可读的 schema 名称
func normalizeSchemaName(s string) string {
	// 移除指针和数组前缀
	s = strings.TrimPrefix(s, "*")
	s = strings.TrimPrefix(s, "[]")
	s = strings.TrimPrefix(s, "*")

	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 2 {
		if normalized := normalizeAliasToDir(parts[0]); normalized != "" {
			return normalized + "." + parts[1]
		}
	}
	return s
}

func sanitizeLabel(label string) string {
	// 从中文标签中提取英文部分作为 schema 名
	// 例如 "重命名 Passkey 凭证" -> "PasskeyRequest"
	words := strings.Fields(label)
	// 收集其中的英文单词
	var englishWords []string
	for _, w := range words {
		hasLetter := false
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			englishWords = append(englishWords, w)
		}
	}
	if len(englishWords) > 0 {
		result := strings.Join(englishWords, "")
		result = strings.ReplaceAll(result, "/", "")
		return result + "Request"
	}
	return "Request"
}

func sanitizeRef(s string) string {
	return normalizeSchemaName(s)
}

func oapiPath(ginPath string) string {
	// Gin 风格的 :param 或 *param 转 OpenAPI 风格 {param}
	// 例如: /account/member/:username -> /account/member/{username}
	parts := strings.Split(ginPath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func addPathItem(oa *OpenAPI, r RouteDef, allSchemas map[string]*SchemaInfo) {
	method := strings.ToLower(r.Method)
	if method == "any" {
		// ANY 路由通常是代理，跳过详细文档
		return
	}

	oapiPath := oapiPath(r.Path)

	if _, exists := oa.Paths[oapiPath]; !exists {
		oa.Paths[oapiPath] = make(map[string]any)
	}

	operation := buildOperation(r, allSchemas)
	oa.Paths[oapiPath][method] = operation
}

// buildResponseSchema 根据 RouteDef 生成响应 schema
func buildResponseSchema(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	// 如果没有指定响应类型，使用默认的 ResponseStruct
	if r.ResponseBodyType == "" {
		return map[string]any{"$ref": "#/components/schemas/" + cfg.ResponseStruct}
	}
	
	// 解析响应类型，生成 data 字段的 schema
	dataSchema := buildDataSchema(r.ResponseBodyType, allSchemas)
	
	// 生成完整的响应 schema（包含 code、message、data）
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type": "integer",
				"description": "状态码",
			},
			"message": map[string]any{
				"type": "string",
				"description": "消息",
			},
			"data": dataSchema,
		},
	}
}

// buildDataSchema 根据 Go 类型字符串生成 data 字段的 schema
func buildDataSchema(typeName string, allSchemas map[string]*SchemaInfo) map[string]any {
	// 处理带包名的 slice 类型，如 account.[]*MemberInfo
	if idx := strings.Index(typeName, ".[]"); idx != -1 {
		pkgName := typeName[:idx]
		elemType := typeName[idx+3:] // 跳过 ".[]" (3个字符)
		elemType = strings.TrimPrefix(elemType, "*") // 去除指针前缀
		
		// 判断是否是基本类型
		if isBasicType(elemType) {
			return map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": getOpenAPIType(elemType),
				},
			}
		}
		
	// 复杂类型，使用 $ref
		// elemType 可能已经包含包名（如 account.MemberInfo），也可能不包含
		schemaName := elemType
		if !strings.Contains(elemType, ".") {
			schemaName = pkgName + "." + elemType
		}
		
		// 检查 schema 是否存在，如果不存在则使用基本类型
		if _, exists := allSchemas[schemaName]; !exists {
			// 尝试不带包名前缀
			if _, exists2 := allSchemas[elemType]; exists2 {
				schemaName = elemType
			} else {
				// schema 不存在，返回基本 object 类型
				return map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"description": "Schema not found: " + schemaName,
					},
				}
			}
		}
		
		return map[string]any{
			"type": "array",
			"items": map[string]any{
				"$ref": "#/components/schemas/" + schemaName,
			},
		}
	}
	
	// 处理不带包名的 slice 类型，如 []*MemberInfo 或 []string
	if strings.HasPrefix(typeName, "[]") {
		elemType := strings.TrimPrefix(typeName, "[]")
		elemType = strings.TrimPrefix(elemType, "*") // 去除指针前缀
		
		// 判断是否是基本类型
		if isBasicType(elemType) {
			return map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": getOpenAPIType(elemType),
				},
			}
		}
		
		// 复杂类型，使用 $ref
		schemaName := strings.ReplaceAll(elemType, ".", "_")
		
		// 检查 schema 是否存在
		if _, exists := allSchemas[schemaName]; !exists {
			// schema 不存在，返回基本 object 类型
			return map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"description": "Schema not found: " + schemaName,
				},
			}
		}
		
		return map[string]any{
			"type": "array",
			"items": map[string]any{
				"$ref": "#/components/schemas/" + schemaName,
			},
		}
	}
	
	// 处理 map 类型
	if strings.HasPrefix(typeName, "map[") {
		return map[string]any{
			"type": "object",
			"description": "Map 类型: " + typeName,
		}
	}
	
	// 处理指针类型
	typeName = strings.TrimPrefix(typeName, "*")

	// 判断是否是基本类型
	if isBasicType(typeName) {
		return map[string]any{
			"type": getOpenAPIType(typeName),
		}
	}

	// 复杂类型，使用 $ref
	// 注意：schema 名称可能包含 "."，如 "account.MemberInfo"
	schemaName := typeName  // 保持原始的包名.类型名格式
	
	// 检查 schema 是否存在
	if _, exists := allSchemas[schemaName]; !exists {
		// schema 不存在，返回基本 object 类型
		return map[string]any{
			"type": "object",
			"description": "Schema not found: " + schemaName,
		}
	}
	
	return map[string]any{
		"$ref": "#/components/schemas/" + schemaName,
	}
}

// isBasicType 判断是否是 Go 基本类型
func isBasicType(typeName string) bool {
	basicTypes := []string{"string", "int", "int8", "int16", "int32", "int64", 
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune"}
	for _, t := range basicTypes {
		if typeName == t {
			return true
		}
	}
	return false
}

// getOpenAPIType 将 Go 类型转换为 OpenAPI 类型
func getOpenAPIType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32":
		return "integer"
	case "int64", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func buildOperation(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	// 生成响应 schema
	responseSchema := buildResponseSchema(r, allSchemas)
	
	op := map[string]any{
		"summary":     r.Label,
		"operationId": generateOperationID(r),
		"tags":        []string{r.Module},
		"responses": map[string]any{
			"200": map[string]any{
				"description": "成功",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": responseSchema,
					},
				},
			},
		},
	}

	// 参数
	var parameters []map[string]any

	// 路径参数
	for _, p := range r.PathParams {
		param := map[string]any{
			"name":     p.Name,
			"in":       "path",
			"required": true,
			"schema": map[string]any{
				"type": p.Type,
			},
		}
		parameters = append(parameters, param)
	}

	// Query 参数（从 struct + 手动提取）
	if r.QueryType != nil {
		for _, f := range r.QueryType.Fields {
			param := map[string]any{
				"name":     f.Name,
				"in":       "query",
				"required": f.Required,
				"schema":   buildProperty(f.Type, allSchemas),
			}
			parameters = append(parameters, param)
		}
	}
	for _, p := range r.QueryParams {
		// 避免与 QueryType 重复
		if r.QueryType != nil {
			dup := false
			for _, f := range r.QueryType.Fields {
				if f.Name == p.Name {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
		}
		param := map[string]any{
			"name":     p.Name,
			"in":       "query",
			"required": false,
			"schema": map[string]any{
				"type": p.Type,
			},
		}
		parameters = append(parameters, param)
	}

	if len(parameters) > 0 {
		op["parameters"] = parameters
	}

	// Request body
	if r.JSONBody != nil && !r.FormData {
		refName := ""
		if r.JSONBody.PkgName != "" {
			refName = sanitizeRef(r.JSONBody.PkgName + "." + r.JSONBody.TypeName)
		} else {
			refName = sanitizeRef(r.JSONBody.TypeName)
		}

		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"$ref": "#/components/schemas/" + refName,
					},
				},
			},
		}
	}

	if r.FormData {
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"multipart/form-data": map[string]any{
					"schema": map[string]any{
						"type": "object",
					},
				},
			},
		}
	}

	if r.IsWS {
		op["description"] = "[WebSocket] " + r.Label
	}

	if r.IsSSE {
		op["description"] = "[SSE] " + r.Label
	}

	return op
}

func generateOperationID(r RouteDef) string {
	// 生成唯一的 operationId
	parts := strings.Split(strings.TrimPrefix(r.Path, "/"), "/")
	var buf strings.Builder
	for i, p := range parts {
		if i > 0 {
			buf.WriteString("_")
		}
		// 移除 :param 和 *param
		p = strings.TrimPrefix(p, ":")
		p = strings.TrimPrefix(p, "*")
		p = strings.ReplaceAll(p, "-", "_")
		buf.WriteString(p)
	}
	return strings.ToLower(r.Method) + "_" + buf.String()
}

// init() 保留给其他初始化逻辑
func init() {
	// APIResponse 的提取已在 main() 中处理
}
