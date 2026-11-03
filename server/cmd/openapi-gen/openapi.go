package main

import (
	"maps"
	"slices"
	"strings"
)

// 整数类型集合（用于 buildProperty 和 extractDefaultQueryParam 的快速判断）
var intTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"byte": true, "rune": true,
}

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
	for _, mod := range slices.Sorted(maps.Keys(modules)) {
		for _, r := range modules[mod] {
			addPathItem(oa, r, allSchemas)
		}
	}

	return oa
}

// ─── 辅助函数 ───

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

// 标准库类型，避免在 buildProperty 中生成无效 $ref
var stdTypes = map[string]bool{
	"time.Time":     true,
	"time.Duration": true,
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
		if _, _, ok := strings.Cut(goType, "."); ok {
			return map[string]any{"$ref": "#/components/schemas/" + sanitizeRef(goType)}
		}
		// 未知类型默认为 object
		return map[string]any{"type": "object"}
	}
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

func buildOperation(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	// 生成响应 schema
	responseSchema := buildResponseSchema(r, allSchemas)
	contentType := r.RawContentType
	if contentType == "" {
		contentType = "application/json"
	}

	op := map[string]any{
		"summary":     r.Label,
		"operationId": generateOperationID(r),
		"tags":        []string{r.Module},
		"responses": map[string]any{
			"200": map[string]any{
				"description": "成功",
				"content": map[string]any{
					contentType: map[string]any{
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

	// Request body（JSON 与 multipart 可同时存在，如 compose 部署两种提交方式都支持）
	if r.JSONBody != nil || r.FormData {
		content := map[string]any{}
		if r.JSONBody != nil {
			name := r.JSONBody.TypeName
			if r.JSONBody.PkgName != "" {
				name = r.JSONBody.PkgName + "." + name
			}
			refName := sanitizeRef(name)
			content["application/json"] = map[string]any{
				"schema": map[string]any{
					"$ref": "#/components/schemas/" + refName,
				},
			}
		}
		if r.FormData {
			content["multipart/form-data"] = map[string]any{
				"schema": buildFormDataSchema(r.FormFields),
			}
		}
		op["requestBody"] = map[string]any{
			"required": true,
			"content":  content,
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

// buildResponseSchema 根据 RouteDef 生成响应 schema
func buildResponseSchema(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	if r.RawContentType != "" {
		return map[string]any{"type": "string"}
	}
	// 如果没有指定响应类型，使用默认的 ResponseStruct
	if r.ResponseBodyType == "" {
		return map[string]any{"$ref": "#/components/schemas/" + cfg.ResponseStruct}
	}

	// 解析响应类型，生成 payload 字段的 schema
	payloadSchema := buildDataSchema(r.ResponseBodyType, allSchemas)

	// 生成完整的响应 schema，与 APIResponse 结构一致
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"success": map[string]any{
				"type":        "boolean",
				"description": "请求是否成功",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "提示信息",
			},
			"payload": payloadSchema,
		},
	}
}

// buildDataSchema 根据 Go 类型字符串生成 data 字段的 schema
func buildDataSchema(typeName string, allSchemas map[string]*SchemaInfo) map[string]any {
	// 处理带包名的 slice 类型，如 account.[]*MemberInfo
	if pkgName, elemType, ok := strings.Cut(typeName, ".[]"); ok {
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
					"type":  "array",
					"items": missingSchema(schemaName),
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
	if after, ok := strings.CutPrefix(typeName, "[]"); ok {
		elemType := after
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
				"type":  "array",
				"items": missingSchema(schemaName),
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
			"type":        "object",
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
	schemaName := typeName // 保持原始的包名.类型名格式

	// 检查 schema 是否存在
	if _, exists := allSchemas[schemaName]; !exists {
		// schema 不存在，返回基本 object 类型
		return missingSchema(schemaName)
	}

	return map[string]any{
		"$ref": "#/components/schemas/" + schemaName,
	}
}

// buildFormDataSchema 根据表单字段构建 multipart/form-data schema
func buildFormDataSchema(fields []ParamDef) map[string]any {
	props := map[string]any{}
	for _, f := range fields {
		switch f.Type {
		case "file":
			props[f.Name] = map[string]any{"type": "string", "format": "binary"}
		case "files":
			props[f.Name] = map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string", "format": "binary"},
			}
		default:
			props[f.Name] = map[string]any{"type": f.Type}
		}
	}
	return map[string]any{"type": "object", "properties": props}
}

// normalizeSchemaName 将 Go import 别名映射为可读的 schema 名称
func normalizeSchemaName(s string) string {
	// 移除指针和数组前缀
	s = strings.TrimPrefix(s, "*")
	s = strings.TrimPrefix(s, "[]")
	s = strings.TrimPrefix(s, "*")

	if alias, name, ok := strings.Cut(s, "."); ok {
		if normalized := normalizeAliasToDir(alias); normalized != "" {
			return normalized + "." + name
		}
	}
	return s
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

// externalSchemas 无法从仓库源码解析的第三方类型（如 libgo 的类型别名）的手写描述。
// 字段需与第三方定义保持一致，升级依赖时同步核对。
var externalSchemas = map[string]map[string]any{
	// github.com/rehiy/libgo/webssh.ListResult
	"webssh.SFTPListResult": {
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "实际目录路径"},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":       map[string]any{"type": "string"},
						"size":       map[string]any{"type": "integer"},
						"mode":       map[string]any{"type": "string"},
						"modTime":    map[string]any{"type": "integer"},
						"isDir":      map[string]any{"type": "boolean"},
						"isLink":     map[string]any{"type": "boolean"},
						"linkTarget": map[string]any{"type": "string"},
					},
				},
			},
		},
	},
}

// missingSchema 为找不到定义的类型生成降级 schema：
// 自由结构（gin.H、map、json.RawMessage）按任意对象处理，已登记的第三方类型使用手写描述，
// 其余保留 "Schema not found" 提示，便于发现真正遗漏的类型。
func missingSchema(schemaName string) map[string]any {
	if ext, ok := externalSchemas[schemaName]; ok {
		return ext
	}
	_, name, _ := strings.Cut(schemaName, ".")
	if schemaName == "gin.H" || schemaName == "json.RawMessage" || name == "map" || strings.HasPrefix(name, "map[") {
		return map[string]any{
			"type":                 "object",
			"description":          "任意 JSON 对象",
			"additionalProperties": true,
		}
	}
	return map[string]any{
		"type":        "object",
		"description": "Schema not found: " + schemaName,
	}
}

// basicTypes 基本类型集合（用于 isBasicType 快速判断）
var basicTypes = map[string]bool{
	"string": true, "bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// isBasicType 判断是否是 Go 基本类型
func isBasicType(typeName string) bool {
	return basicTypes[typeName]
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
