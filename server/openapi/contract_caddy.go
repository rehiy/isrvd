package openapi

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// completeBasicAuthContract 补充按 snapshot 分支返回的类型和显式版本校验；
// binding:"required" 会使缺失版本返回 400，不能用于表达实际的 428 契约。
func (g *generator) completeBasicAuthContract(oa *OpenAPI, allSchemas map[string]*SchemaInfo) {
	for _, r := range g.routes {
		handler := extractHandlerName(r.Handler)
		switch handler {
		case "caddyBasicAuthList", "caddyBasicAuthUserCreate", "caddyBasicAuthUserDelete", "caddyBasicAuthConfigUpdate":
		default:
			continue
		}
		op := oa.Paths[oapiPath(r.Path)][strings.ToLower(r.Method)].(map[string]any)
		responses := op["responses"].(map[string]any)
		if handler == "caddyBasicAuthList" {
			routesSchema := buildDataSchema("caddy.[]BasicAuthRouteView", allSchemas)
			routesSchema["nullable"] = true
			responseSchema := g.buildResponseSchema(r, allSchemas)
			responseSchema["properties"].(map[string]any)["payload"] = map[string]any{
				"oneOf": []any{routesSchema, buildDataSchema("caddy.BasicAuthSnapshot", allSchemas)},
			}
			responses["200"].(map[string]any)["content"].(map[string]any)["application/json"] = map[string]any{"schema": responseSchema}
			for _, name := range []string{"caddy.BasicAuthRouteView", "caddy.BasicAuthSnapshot"} {
				schema := oa.Components.Schemas[name].(map[string]any)
				schema["properties"].(map[string]any)["revision"].(map[string]any)["readOnly"] = true
			}
			snapshot := oa.Components.Schemas["caddy.BasicAuthSnapshot"].(map[string]any)
			snapshotRoutes := snapshot["properties"].(map[string]any)["routes"].(map[string]any)
			maps.Copy(snapshotRoutes, routesSchema)
			continue
		}

		if r.JSONBody != nil {
			schemaName := normalizeSchemaName(r.JSONBody.PkgName + "." + r.JSONBody.TypeName)
			schema := oa.Components.Schemas[schemaName].(map[string]any)
			required, _ := schema["required"].([]string)
			if !slices.Contains(required, "revision") {
				schema["required"] = append(required, "revision")
			}
		}
		if handler == "caddyBasicAuthUserDelete" {
			for _, param := range op["parameters"].([]map[string]any) {
				if param["in"] == "query" && param["name"] == "revision" {
					param["required"] = true
				}
			}
		}
		errors := map[int]string{
			http.StatusPreconditionRequired: "缺少路由快照 revision，不加载配置",
			http.StatusConflict:             "服务路由快照已变化，不加载配置",
		}
		if handler == "caddyBasicAuthConfigUpdate" {
			errors[http.StatusForbidden] = "移除账号时缺少 DELETE 权限，不加载配置"
		}
		for status, description := range errors {
			responses[strconv.Itoa(status)] = map[string]any{
				"description": description,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/" + g.cfg.ResponseStruct},
					},
				},
			}
		}
	}
}
