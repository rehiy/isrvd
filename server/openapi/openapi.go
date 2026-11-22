package openapi

import (
	"maps"
	"slices"
	"strings"
)

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

func (g *generator) buildOpenAPI() (*OpenAPI, error) {
	oa := &OpenAPI{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:   g.cfg.Title,
			Version: g.cfg.Version,
		},
		Servers: []OpenAPIServer{
			{URL: g.cfg.ServerURL, Description: g.cfg.ServerDesc},
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
	allSchemas := g.collectAllSchemas()
	for name, schema := range allSchemas {
		schemaDef := buildSchemaObject(schema, allSchemas)
		oa.Components.Schemas[name] = schemaDef
	}

	for name, schema := range externalSchemas {
		if _, exists := oa.Components.Schemas[name]; !exists {
			oa.Components.Schemas[name] = schema
		}
	}

	// 按模块分组路由
	modules := make(map[string][]RouteDef)
	for _, r := range g.routes {
		modules[r.Module] = append(modules[r.Module], r)
	}

	// 排序模块名
	for _, mod := range slices.Sorted(maps.Keys(modules)) {
		for _, r := range modules[mod] {
			g.addPathItem(oa, r, allSchemas)
		}
	}

	g.completeBasicAuthContract(oa, allSchemas)
	if err := oa.retainReferencedSchemas(); err != nil {
		return nil, err
	}
	return oa, nil
}

func (g *generator) addPathItem(oa *OpenAPI, r RouteDef, allSchemas map[string]*SchemaInfo) {
	method := strings.ToLower(r.Method)
	if method == "any" {
		// ANY 路由通常是代理，跳过详细文档
		return
	}

	oapiPath := oapiPath(r.Path)

	if _, exists := oa.Paths[oapiPath]; !exists {
		oa.Paths[oapiPath] = make(map[string]any)
	}

	operation := g.buildOperation(r, allSchemas)
	oa.Paths[oapiPath][method] = operation
}

func (g *generator) buildOperation(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	// 生成响应 schema
	responseSchema := g.buildResponseSchema(r, allSchemas)
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
	for _, part := range strings.Split(r.Path, "/") {
		if !strings.HasPrefix(part, ":") && !strings.HasPrefix(part, "*") {
			continue
		}
		param := map[string]any{
			"name":     part[1:],
			"in":       "path",
			"required": true,
			"schema": map[string]any{
				"type": "string",
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
			refName := normalizeSchemaName(name)
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
func (g *generator) buildResponseSchema(r RouteDef, allSchemas map[string]*SchemaInfo) map[string]any {
	if r.RawContentType == "application/octet-stream" {
		return map[string]any{"type": "string", "format": "binary"}
	}
	if r.RawContentType != "" {
		return map[string]any{"type": "string"}
	}
	// 如果没有指定响应类型，使用默认的 ResponseStruct
	if r.ResponseBodyType == "" {
		return map[string]any{"$ref": "#/components/schemas/" + g.cfg.ResponseStruct}
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
