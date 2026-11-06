package openapi

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

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

// collectServiceTypes 遍历 service 及额外类型目录，解析所有类型定义
func (g *generator) collectServiceTypes() {
	// 遍历 service 目录下的各个包子目录
	serviceDir := filepath.Join(g.root, g.cfg.ServiceDir)
	entries, err := os.ReadDir(serviceDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				g.collectTypesFromDir(filepath.Join(serviceDir, entry.Name()))
			}
		}
	}

	// 遍历额外的类型定义目录
	for _, dir := range g.cfg.TypeDirs {
		g.collectTypesFromDir(filepath.Join(g.root, dir))
	}
}

// collectTypesFromDir 遍历目录，解析所有 Go 文件中的类型定义
func (g *generator) collectTypesFromDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// 递归遍历子目录
			subDir := filepath.Join(dir, entry.Name())
			g.collectTypesFromDir(subDir)
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		g.collectTypeDefinitionsFromFile(filename)
	}
}

// collectTypeDefinitionsFromFile 解析文件中的顶层类型定义
func (g *generator) collectTypeDefinitionsFromFile(filename string) {
	f := g.parseFile(filename)
	if f == nil {
		return
	}

	// 从文件路径提取包名
	pkgName := g.extractPkgNameFromPath(filename)

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

			// 只收集 struct 类型，跳过 interface、type alias 等
			st, isStruct := ts.Type.(*ast.StructType)
			if !isStruct {
				continue
			}

			typeName := ts.Name.Name

			// 跳过非导出类型和 Service/Client 等内部实现类型
			if len(typeName) == 0 || typeName[0] < 'A' || typeName[0] > 'Z' {
				continue
			}

			// 预收集仅包含 JSON 字段的类型；接口使用的空类型由 handler 分析按需解析。
			if len(g.buildSchemaFromStruct(pkgName, typeName, st, filename).Fields) == 0 {
				continue
			}
			// 运行时配置快照只用于内部持久化和快照传递，不属于公开 API 契约。
			if pkgName == "config" && typeName == "Snapshot" {
				continue
			}

			// 构建完整的类型名 (包名.类型名)
			if pkgName != "" {
				g.resolveStructSchema(pkgName+"."+typeName, nil, filename)
			}
		}
	}
}

func (g *generator) collectAllTypesFromFile(filename string) {
	f := g.parseFile(filename)
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
								g.resolveStructSchema(typeName, nil, filename)
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
							g.resolveStructSchema(typeName, nil, filename)
						}
					}
				}
				return true
			})
			return true
		})
	}
}

func (g *generator) resolveStructSchema(typeName string, localTypes map[string]string, ctrlFile string) *SchemaInfo {
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
				// 优先从源文件提取完整 schema（含注释）
				if ctrlFile != "" {
					if schema := g.findStructInFile(ctrlFile, name, ""); schema != nil {
						g.structCache[name] = schema
						g.resolveNestedTypesInSchema(schema, ctrlFile)
						return schema
					}
				}
				// 降级：使用序列化的字段信息（无注释）
				info := parseFieldsString(name, fields)
				if info != nil {
					cacheKey := name
					g.structCache[cacheKey] = info
					g.resolveNestedTypesInSchema(info, ctrlFile)
					return info
				}
			}
		}
	}

	// 3. 检查是否为带包名的类型
	pkgAlias, structName, hasPkg := strings.Cut(typeName, ".")
	if !hasPkg {
		pkgAlias, structName = "", typeName
	}

	// 4. 从缓存中查找
	cacheKey := pkgAlias + "." + structName
	if cached, ok := g.structCache[cacheKey]; ok {
		return cached
	}

	// 5. 解析对应的 service 文件
	schema := g.parseServiceStruct(pkgAlias, structName, ctrlFile)
	if schema != nil {
		// 使用规范化名称作为缓存 key
		normalizedKey := normalizeSchemaName(cacheKey)
		g.structCache[normalizedKey] = schema
		// 同时也用原始 key
		g.structCache[cacheKey] = schema

		// 6. 递归解析嵌套结构体字段类型（如 *config.CopilotConfig）
		g.resolveNestedTypesInSchema(schema, ctrlFile)
	}
	return schema
}

// resolveNestedTypesInSchema 解析 schema 字段中引用的嵌套结构体类型
func (g *generator) resolveNestedTypesInSchema(schema *SchemaInfo, ctrlFile string) {
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
		g.resolveStructSchema(goType, nil, ctrlFile)
	}
}

func (g *generator) parseServiceStruct(pkgAlias, structName string, ctrlFile string) *SchemaInfo {
	serviceDir := filepath.Join(g.root, g.cfg.ServiceDir)

	// 构建搜索目录列表
	searchDirs := g.buildSearchDirs(serviceDir, pkgAlias)

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
			schema := g.findStructInFile(filename, structName, pkgAlias)
			if schema != nil {
				return schema
			}
		}
	}
	return nil
}

// findStructInFile 在单个文件中查找结构体定义
func (g *generator) findStructInFile(filename, structName, pkgAlias string) *SchemaInfo {
	f := g.parseFile(filename)
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
			return g.buildSchemaFromStruct(pkgAlias, structName, st, filename)
		}
	}
	return nil
}

// buildSchemaFromStruct 从 AST StructType 构建 SchemaInfo
func (g *generator) buildSchemaFromStruct(pkgAlias, structName string, st *ast.StructType, filename string) *SchemaInfo {
	schema := &SchemaInfo{
		PkgName:  pkgAlias,
		TypeName: structName,
	}

	// 读取文件内容以获取字段注释
	fieldComments := make(map[string]string)
	if filename != "" {
		fieldComments = g.extractFieldComments(filename, st)
	}

	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue // embedded
		}
		for _, name := range field.Names {
			// 跳过未导出字段：encoding/json 不会序列化它们，绝非 HTTP 输入输出参数
			if !name.IsExported() {
				continue
			}
			// 提取字段信息
			fieldInfo := extractFieldInfo(name.Name, field)
			// 优先使用字段注释，其次使用 json tag 中的 description
			if comment, ok := fieldComments[name.Name]; ok && comment != "" {
				fieldInfo.Description = comment
			}
			// 跳过字段名为 "-" 的字段
			if fieldInfo.Name != "-" {
				schema.Fields = append(schema.Fields, fieldInfo)
			}
		}
	}
	return schema
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
		// 内联结构体的字段描述：优先行尾注释，其次字段上方注释（AST 已带注释解析）
		comment := extractComment(field.Comment)
		if comment == "" {
			comment = extractComment(field.Doc)
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			fieldInfo := extractFieldInfo(name.Name, field)
			if comment != "" && fieldInfo.Description == "" {
				fieldInfo.Description = comment
			}
			if fieldInfo.Name != "-" {
				schema.Fields = append(schema.Fields, fieldInfo)
			}
		}
	}
	return schema
}

func structTypeToJSONSchema(st *ast.StructType) string {
	if st.Fields == nil {
		return ""
	}
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

// ─── 辅助函数 ───

// extractPkgNameFromPath 从文件路径提取包名
// 根据配置的目录结构动态推断包名
func (g *generator) extractPkgNameFromPath(filename string) string {
	// 处理 service 目录（如 service/account/member.go → account）
	svcMarker := "/" + filepath.ToSlash(g.cfg.ServiceDir) + "/"
	if _, after, ok := strings.Cut(filename, svcMarker); ok {
		rest := after
		subDir, _, _ := strings.Cut(rest, "/")
		return subDir
	}

	// 处理额外的类型目录
	for _, dir := range g.cfg.TypeDirs {
		marker := "/" + filepath.ToSlash(dir) + "/"
		if _, after, ok := strings.Cut(filename, marker); ok {
			rest := after
			// 如果 rest 中还有子目录，取第一层作为包名
			if subDir, _, _ := strings.Cut(rest, "/"); !strings.HasSuffix(subDir, ".go") {
				return subDir
			}
			// 否则直接用目录名作为包名（如 config/types.go → config）
			return filepath.Base(dir)
		}
	}

	return ""
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

// buildSearchDirs 构建需要搜索的目录列表
func (g *generator) buildSearchDirs(serviceDir, pkgAlias string) []string {
	var dirs []string
	if pkgAlias != "" {
		dirs = append(dirs, filepath.Join(serviceDir, pkgAlias))
		// 额外类型目录
		for _, td := range g.cfg.TypeDirs {
			dirs = append(dirs, filepath.Join(g.root, td, pkgAlias))
		}
		dirs = append(dirs, filepath.Join(g.root, pkgAlias)) // 根级包
		// 类型目录自身即包目录（如 server/config → config）
		for _, td := range g.cfg.TypeDirs {
			if filepath.Base(td) == pkgAlias {
				dirs = append(dirs, filepath.Join(g.root, td))
			}
		}
	}

	if normalized := normalizeAliasToDir(pkgAlias); normalized != "" && normalized != pkgAlias {
		dirs = append(dirs, filepath.Join(serviceDir, normalized))
		for _, td := range g.cfg.TypeDirs {
			dirs = append(dirs, filepath.Join(g.root, td, normalized))
		}
		dirs = append(dirs, filepath.Join(g.root, normalized)) // 根级包
	}
	return dirs
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

// extractFieldComments 从结构体定义中提取字段注释
// 支持三种格式：
//  1. 字段上方注释：// 字段描述
//  2. 字段后面注释：FieldName Type `json:"..."` // 字段描述
//  3. tag 中的 description："description" 字段描述
func (g *generator) extractFieldComments(filename string, st *ast.StructType) map[string]string {
	comments := make(map[string]string)

	// 使用缓存的文件内容
	lines := g.getFileLines(filename)
	if lines == nil {
		return comments
	}

	// 遍历结构体字段，查找注释
	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue
		}

		// 获取字段所在的行号（AST 行号从 1 开始）
		fieldLine := g.fset.Position(field.Pos()).Line

		// 1. 检查字段所在行的行内注释（trailing comment）
		if fieldLine <= len(lines) {
			lineText := lines[fieldLine-1] // 数组索引从 0 开始
			if _, after, ok := strings.Cut(lineText, "//"); ok {
				commentText := strings.TrimSpace(after)
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
						if after, ok := strings.CutPrefix(commentText, name.Name+" "); ok {
							// 格式："FieldName 描述"
							comments[name.Name] = after
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
	for field := range strings.SplitSeq(s, ";") {
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
