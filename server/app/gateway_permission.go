package app

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"isrvd/server/service/node"
)

// initGatewayPermissions 建立网关的权限匹配索引，不执行任何业务 handler。
// 本机业务路由取自 app，节点管理路由取自 service/node，账户权限列表共用相同元信息。
func (app *App) initGatewayPermissions() {
	app.gatewayPermissions = http.NewServeMux()
	app.gatewayRouteIndex = make(map[string]Route)
	if app.mode == ModeCenter {
		for _, entry := range node.PermissionRoutes() {
			key := entry.Method + " " + entry.Path
			app.routeIndex[key] = Route{Key: key, Method: entry.Method, Path: strings.TrimPrefix(entry.Path, APINamespace), Module: "node", Label: entry.Label}
		}
	}
	for _, route := range app.routeIndex {
		parts := strings.Split(APINamespace+route.Path, "/")
		for i, part := range parts {
			if strings.HasPrefix(part, ":") {
				parts[i] = "{" + part[1:] + "}"
			}
			if strings.HasPrefix(part, "*") {
				parts[i] = "{" + part[1:] + "...}"
			}
		}
		pattern := strings.Join(parts, "/")
		if route.Method != "ANY" {
			pattern = route.Method + " " + pattern
		}
		app.gatewayPermissions.HandleFunc(pattern, func(http.ResponseWriter, *http.Request) {})
		app.gatewayRouteIndex[pattern] = route
	}
}

// GatewayPermission 校验转发请求的规范路由权限，读取当前账户服务，配置重载后自动生效。
// 不根据中控本机的服务可用性阻止节点业务请求，实际服务由受管机提供。
func GatewayPermission(r *http.Request, username string) (int, error) {
	app := running.Load()
	if app == nil || !app.servicesMu.TryRLock() {
		return http.StatusBadGateway, errGatewayUnavailable
	}
	defer app.servicesMu.RUnlock()
	app.requestCtxMu.RLock()
	ready := app.requestsCtx != nil && app.requestsCtx.Err() == nil
	app.requestCtxMu.RUnlock()
	if !ready || app.mode != ModeCenter || app.accountSvc == nil || app.gatewayPermissions == nil {
		return http.StatusBadGateway, errGatewayUnavailable
	}
	// 与 Gin 默认使用解码后 URL.Path 的行为一致，禁止清理路径或转义分段改变匹配结果。
	if path.Clean(r.URL.Path) != r.URL.Path {
		return http.StatusForbidden, fmt.Errorf("未授权的访问路径")
	}
	lookup := r.Clone(r.Context())
	lookup.URL.RawPath = ""
	_, pattern := app.gatewayPermissions.Handler(lookup)
	route, ok := app.gatewayRouteIndex[pattern]
	if !ok || (route.Method != "ANY" && route.Method != r.Method) {
		return http.StatusForbidden, fmt.Errorf("未授权的访问路径")
	}
	if route.Access == AccessPerm && route.Module != "" {
		if err := app.accountSvc.PermCheck(username, route.Label, r.Method, APINamespace+route.Path); err != nil {
			return http.StatusForbidden, err
		}
	}
	return http.StatusOK, nil
}
