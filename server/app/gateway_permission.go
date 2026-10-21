package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"isrvd/server/service/account"
	"isrvd/server/service/node"
)

var errNotLoggedIn = errors.New("未登录或登录已过期")

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
		pattern := account.MuxPattern(route.Method, APINamespace+route.Path)
		app.gatewayPermissions.HandleFunc(pattern, func(http.ResponseWriter, *http.Request) {})
		app.gatewayRouteIndex[pattern] = route
	}
}

// GatewayAuthorize 为中控网关认证调用者并校验请求的规范路由权限，返回用户名；
// 失败时同时返回应写出的 HTTP 状态码。认证与权限在同一次生命周期加锁内完成，读取当前账户服务，配置重载后自动生效。
// 网关须先校验 THA 来源并清理外部身份头。
//
// 应用未就绪或正在重载返回 502，而不是 401：否则前端会把一次上游抖动当成登录过期而清掉登录态。
// 不根据中控本机的服务可用性阻止节点业务请求，实际服务由受管机提供。
func GatewayAuthorize(r *http.Request, allowQueryToken bool) (string, int, error) {
	app := running.Load()
	if app == nil || !app.servicesMu.TryRLock() {
		return "", http.StatusBadGateway, errGatewayUnavailable
	}
	defer app.servicesMu.RUnlock()
	app.requestCtxMu.RLock()
	ready := app.requestsCtx != nil && app.requestsCtx.Err() == nil
	app.requestCtxMu.RUnlock()
	if !ready || app.mode != ModeCenter || app.accountSvc == nil || app.gatewayPermissions == nil {
		return "", http.StatusBadGateway, errGatewayUnavailable
	}

	username := app.accountSvc.GatewayUser(r, allowQueryToken)
	if username == "" {
		return "", http.StatusUnauthorized, errNotLoggedIn
	}

	// 与 Gin 默认使用解码后 URL.Path 的行为一致，禁止清理路径或转义分段改变匹配结果。
	if path.Clean(r.URL.Path) != r.URL.Path {
		return "", http.StatusForbidden, fmt.Errorf("未授权的访问路径")
	}
	// 匹配只需要方法、主机与解码后的路径；不复制整个请求（头部、上下文等）。
	lookup := &http.Request{Method: r.Method, Host: r.Host, URL: &url.URL{Path: r.URL.Path}}
	_, pattern := app.gatewayPermissions.Handler(lookup)
	route, ok := app.gatewayRouteIndex[pattern]
	if !ok || (route.Method != "ANY" && route.Method != r.Method) {
		return "", http.StatusForbidden, fmt.Errorf("未授权的访问路径")
	}
	if route.Access == AccessPerm && route.Module != "" {
		if err := app.accountSvc.PermCheck(username, route.Label, r.Method, APINamespace+route.Path); err != nil {
			return "", http.StatusForbidden, err
		}
	}
	return username, http.StatusOK, nil
}
