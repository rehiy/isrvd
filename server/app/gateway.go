package app

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	"isrvd/pkgs/helper"
	"isrvd/server/service/node"
)

// 本文件是 app 面向中控网关的全部入口：网关所需的路由信息、认证与权限校验。
// 仅中控模式使用；其他模式不构建权限索引，也不会被网关调用。

var (
	errNotLoggedIn   = errors.New("未登录或登录已过期")
	errForbiddenPath = errors.New("未授权的访问路径")
)

// centerModules 节点视角下仍由中控处理、不转发给受管机的模块：账号体系与 AI 助手始终作用于中控。
var centerModules = []string{"account", "copilot"}

// GatewayRoutes 从路由定义中提取中控网关需要的信息（只收集一次路由表）：
//   - queryToken：允许通过 ?token= 认证的 GET 路由（Gin 路径格式，含 /api 前缀），
//     网关据此对转发到节点的请求套用与 isrvd 相同的规则，避免令牌随意出现在 URL 中；
//   - nodeModules：节点视角下转发给受管机的业务模块（即 /api/<模块>/ 的首段），
//     即所有路由所在的模块去掉 centerModules。
func GatewayRoutes() (queryToken, nodeModules []string) {
	for _, route := range (&App{}).collectRoutes() {
		if route.QueryToken && route.Method == http.MethodGet {
			queryToken = append(queryToken, APINamespace+route.Path)
		}
		module := strings.SplitN(route.Path, "/", 3)[1]
		if !slices.Contains(centerModules, module) && !slices.Contains(nodeModules, module) {
			nodeModules = append(nodeModules, module)
		}
	}
	return queryToken, nodeModules
}

// initGatewayPermissions 建立网关的权限匹配索引，不执行任何业务 handler。
// 本机业务路由取自 app，节点管理路由取自 service/node，账户权限列表共用相同元信息。
func (app *App) initGatewayPermissions() {
	app.gatewayPermissions = http.NewServeMux()
	app.gatewayRouteIndex = make(map[string]Route)
	for _, entry := range node.PermissionRoutes() {
		key := entry.Method + " " + entry.Path
		app.routeIndex[key] = Route{Key: key, Method: entry.Method, Path: strings.TrimPrefix(entry.Path, APINamespace), Module: "node", Label: entry.Label}
	}
	for _, route := range app.routeIndex {
		pattern := helper.MuxPattern(route.Method, APINamespace+route.Path)
		app.gatewayPermissions.HandleFunc(pattern, func(http.ResponseWriter, *http.Request) {})
		app.gatewayRouteIndex[pattern] = route
	}
}

// GatewayAuthorize 为中控网关认证调用者并校验请求的规范路由权限，返回用户名；
// 失败时同时返回应写出的 HTTP 状态码。认证与权限在同一次生命周期加锁内完成，读取当前账户服务，配置重载后自动生效。
// 网关须先通过 account.TrustGatewayIdentity 校验 THA 来源并清理外部身份头。
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
	if !ready || app.accountSvc == nil || app.gatewayPermissions == nil {
		return "", http.StatusBadGateway, errGatewayUnavailable
	}

	username := app.accountSvc.GatewayUser(r, allowQueryToken)
	if username == "" {
		return "", http.StatusUnauthorized, errNotLoggedIn
	}

	// 与 Gin 默认使用解码后 URL.Path 的行为一致，禁止清理路径或转义分段改变匹配结果。
	if path.Clean(r.URL.Path) != r.URL.Path {
		return "", http.StatusForbidden, errForbiddenPath
	}
	// 匹配只需要方法、主机与解码后的路径；不复制整个请求（头部、上下文等）。
	lookup := &http.Request{Method: r.Method, Host: r.Host, URL: &url.URL{Path: r.URL.Path}}
	_, pattern := app.gatewayPermissions.Handler(lookup)
	route, ok := app.gatewayRouteIndex[pattern]
	if !ok || (route.Method != "ANY" && route.Method != r.Method) {
		return "", http.StatusForbidden, errForbiddenPath
	}
	if route.Access == AccessPerm && route.Module != "" {
		if err := app.accountSvc.PermCheck(username, route.Label, r.Method, APINamespace+route.Path); err != nil {
			return "", http.StatusForbidden, err
		}
	}
	return username, http.StatusOK, nil
}
