package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	pkgCaddy "isrvd/pkgs/caddy"
	"isrvd/server/service/caddy"
)

// defineCaddyRoutes 定义 Caddy 模块路由
func (app *App) defineCaddyRoutes() []Route {
	return []Route{
		// 概览与原始配置
		{Method: "GET", Path: "/caddy/info", Handler: app.caddyInfoInspect, Module: "caddy", Label: "查询 Caddy 概览"},
		{Method: "GET", Path: "/caddy/config", Handler: app.caddyConfigInspect, Module: "caddy", Label: "查询 Caddy 完整配置"},
		{Method: "POST", Path: "/caddy/config", Handler: app.caddyConfigLoad, Module: "caddy", Label: "整体替换 Caddy 配置"},
		// 指标开关（抓取出口见 /metrics/caddy）
		{Method: "PUT", Path: "/caddy/metrics", Handler: app.caddyMetricsUpdate, Module: "caddy", Label: "开关 Caddy 指标"},
		// 全局选项
		{Method: "GET", Path: "/caddy/global", Handler: app.caddyGlobalInspect, Module: "caddy", Label: "查询 Caddy 全局选项"},
		{Method: "PUT", Path: "/caddy/global", Handler: app.caddyGlobalUpdate, Module: "caddy", Label: "更新 Caddy 全局选项"},
		// 路由 CRUD（默认 server=srv0，可通过 query 指定）
		{Method: "GET", Path: "/caddy/routes", Handler: app.caddyRouteList, Module: "caddy", Label: "查询 Caddy 路由列表"},
		{Method: "GET", Path: "/caddy/route/:index", Handler: app.caddyRouteInspect, Module: "caddy", Label: "获取 Caddy 路由详情"},
		{Method: "POST", Path: "/caddy/route", Handler: app.caddyRouteCreate, Module: "caddy", Label: "创建 Caddy 路由"},
		{Method: "PUT", Path: "/caddy/route/:index", Handler: app.caddyRouteUpdate, Module: "caddy", Label: "更新 Caddy 路由"},
		{Method: "PUT", Path: "/caddy/route/:index/position", Handler: app.caddyRouteMove, Module: "caddy", Label: "调整 Caddy 路由顺序"},
		{Method: "DELETE", Path: "/caddy/route/:index", Handler: app.caddyRouteDelete, Module: "caddy", Label: "删除 Caddy 路由"},
		// HTTP server CRUD
		{Method: "GET", Path: "/caddy/servers", Handler: app.caddyServerList, Module: "caddy", Label: "查询 Caddy 服务列表"},
		{Method: "GET", Path: "/caddy/server/:name", Handler: app.caddyServerInspect, Module: "caddy", Label: "获取 Caddy 服务详情"},
		{Method: "POST", Path: "/caddy/server", Handler: app.caddyServerCreate, Module: "caddy", Label: "创建 Caddy 服务"},
		{Method: "PUT", Path: "/caddy/server/:name", Handler: app.caddyServerUpdate, Module: "caddy", Label: "更新 Caddy 服务"},
		{Method: "DELETE", Path: "/caddy/server/:name", Handler: app.caddyServerDelete, Module: "caddy", Label: "删除 Caddy 服务"},
		// Basic Auth 账号管理
		{Method: "GET", Path: "/caddy/basic-auth", Handler: app.caddyBasicAuthList, Module: "caddy", Label: "查询 Caddy Basic Auth 路由列表"},
		{Method: "POST", Path: "/caddy/basic-auth/:index/users", Handler: app.caddyBasicAuthUserCreate, Module: "caddy", Label: "添加 Caddy Basic Auth 账号"},
		{Method: "DELETE", Path: "/caddy/basic-auth/:index/users/:username", Handler: app.caddyBasicAuthUserDelete, Module: "caddy", Label: "删除 Caddy Basic Auth 账号"},
		{Method: "PUT", Path: "/caddy/basic-auth/:index/config", Handler: app.caddyBasicAuthConfigUpdate, Module: "caddy", Label: "更新 Caddy Basic Auth 配置"},
		// SSL 证书 CRUD
		{Method: "GET", Path: "/caddy/certs", Handler: app.caddyCertList, Module: "caddy", Label: "查询 Caddy SSL 证书列表"},
		{Method: "POST", Path: "/caddy/cert", Handler: app.caddyCertCreate, Module: "caddy", Label: "创建 Caddy SSL 证书"},
		{Method: "PUT", Path: "/caddy/cert/:key", Handler: app.caddyCertUpdate, Module: "caddy", Label: "更新 Caddy SSL 证书"},
		{Method: "DELETE", Path: "/caddy/cert/:key", Handler: app.caddyCertDelete, Module: "caddy", Label: "删除 Caddy SSL 证书"},
	}
}

func (app *App) caddyMetricsUpdate(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled" binding:"required"` // true 开启，false 关闭
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondMetricsToggled(c, *req.Enabled, app.availability.Load().caddySvc.MetricsUpdate(c.Request.Context(), *req.Enabled))
}

// ─── HTTP server CRUD ───

// caddyServerURI 的 :name 实为服务 id（`~` + base64url(name)，也兼容原始名称）
type caddyServerURI struct {
	Name string `uri:"name" binding:"required"`
}

func (app *App) caddyServerList(c *gin.Context) {
	result, err := app.availability.Load().caddySvc.ServerList(c.Request.Context())
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyServerInspect(c *gin.Context) {
	var uri caddyServerURI
	if err := c.ShouldBindUri(&uri); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().caddySvc.ServerInspect(c.Request.Context(), uri.Name)
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyServerCreate(c *gin.Context) {
	var req caddy.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.ServerCreate(c.Request.Context(), req); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 服务创建成功", nil)
}

func (app *App) caddyServerUpdate(c *gin.Context) {
	var uri caddyServerURI
	if err := c.ShouldBindUri(&uri); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req caddy.ServerForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.ServerUpdate(c.Request.Context(), uri.Name, req); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 服务更新成功", nil)
}

func (app *App) caddyServerDelete(c *gin.Context) {
	var uri caddyServerURI
	if err := c.ShouldBindUri(&uri); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.ServerDelete(c.Request.Context(), uri.Name); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 服务删除成功", nil)
}

// ─── 概览与原始配置 ───

func (app *App) caddyInfoInspect(c *gin.Context) {
	result, err := app.availability.Load().caddySvc.Info(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) caddyConfigInspect(c *gin.Context) {
	result, err := app.availability.Load().caddySvc.ConfigAll(c.Request.Context())
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyConfigLoad(c *gin.Context) {
	var req struct {
		Config json.RawMessage `json:"config" binding:"required"` // 完整的 Caddy JSON 配置
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.ConfigLoad(c.Request.Context(), req.Config); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "Caddy 配置已加载", nil)
}

// ─── 全局选项 ───

func (app *App) caddyGlobalInspect(c *gin.Context) {
	result, err := app.availability.Load().caddySvc.Global(c.Request.Context())
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyGlobalUpdate(c *gin.Context) {
	var req caddy.GlobalForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.GlobalUpdate(c.Request.Context(), req); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "全局选项已更新", nil)
}

// ─── 路由 CRUD ───

func (app *App) caddyRouteList(c *gin.Context) {
	var query struct {
		Server   string `form:"server"`
		Snapshot bool   `form:"snapshot"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, revision, err := app.availability.Load().caddySvc.RouteList(c.Request.Context(), query.Server)
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	if query.Snapshot {
		respondSuccess(c, "", caddy.RouteSnapshot{Routes: result, Revision: revision})
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyRouteInspect(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	result, err := app.availability.Load().caddySvc.RouteInspect(c.Request.Context(), server, idx)
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyRouteCreate(c *gin.Context) {
	var query struct {
		Server   string `form:"server"`
		Index    *int   `form:"index"`
		Revision string `form:"revision"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req pkgCaddy.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	idx, err := app.availability.Load().caddySvc.RouteCreate(c.Request.Context(), query.Server, req, query.Index, query.Revision)
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 路由创建成功", gin.H{"index": idx})
}

func (app *App) caddyRouteUpdate(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	var req pkgCaddy.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.RouteUpdate(c.Request.Context(), server, idx, req, c.Query("revision")); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 路由更新成功", nil)
}

func (app *App) caddyRouteDelete(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	if err := app.availability.Load().caddySvc.RouteDelete(c.Request.Context(), server, idx, c.Query("revision")); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 路由删除成功", nil)
}

func (app *App) caddyRouteMove(c *gin.Context) {
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	var req caddy.RouteMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.RouteMove(c.Request.Context(), c.Query("server"), idx, req); err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "Caddy 路由顺序已更新", nil)
}

// ─── Basic Auth 账号管理 ───

func (app *App) caddyBasicAuthList(c *gin.Context) {
	var query struct {
		Server   string `form:"server"`
		Snapshot bool   `form:"snapshot"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().caddySvc.BasicAuthSnapshot(c.Request.Context(), query.Server)
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	if query.Snapshot {
		respondSuccess(c, "", result)
		return
	}
	respondSuccess(c, "", result.Routes)
}

func (app *App) caddyBasicAuthUserCreate(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	var req struct {
		Username      string `json:"username"      binding:"required"` // basic_auth 登录用户名
		Password      string `json:"password"      binding:"required"` // 明文密码，由后端转换为 bcrypt hash
		Realm         string `json:"realm"`                            // HTTP Basic realm；留空：已有 handler 沿用原 realm，新建使用 Caddy 默认值
		ForwardHeader string `json:"forwardHeader"`                    // 认证成功后传递用户名的请求头；留空不转发
		Revision      string `json:"revision"`                         // 必填：所属服务路由列表的快照版本；缺失返回 428
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Revision == "" {
		respondCaddyError(c, caddy.ErrRouteRevisionRequired)
		return
	}
	if err := app.availability.Load().caddySvc.BasicAuthUserCreate(c.Request.Context(), server, idx, req.Username, req.Password, req.Realm, req.ForwardHeader, req.Revision); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "账号添加成功", nil)
}

func (app *App) caddyBasicAuthUserDelete(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	revision := c.Query("revision")
	if revision == "" {
		respondCaddyError(c, caddy.ErrRouteRevisionRequired)
		return
	}
	if err := app.availability.Load().caddySvc.BasicAuthUserDelete(c.Request.Context(), server, idx, c.Param("username"), revision); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "账号删除成功", nil)
}

func (app *App) caddyBasicAuthConfigUpdate(c *gin.Context) {
	server := c.Query("server")
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "index 必须为整数")
		return
	}
	var req struct {
		Realm         string   `json:"realm"`         // HTTP Basic realm；留空使用 Caddy 默认值
		ForwardHeader string   `json:"forwardHeader"` // 认证成功后传递用户名的请求头；留空不转发
		Revision      string   `json:"revision"`      // 必填：所属服务路由快照版本；缺失返回 428
		RemoveUsers   []string `json:"removeUsers"`   // 原子移除账号；另需 DELETE 权限，至少保留一个账号
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Revision == "" {
		respondCaddyError(c, caddy.ErrRouteRevisionRequired)
		return
	}
	if len(req.RemoveUsers) > 0 {
		if err := app.accountSvc.PermCheck(c.GetString("username"), "删除 Caddy Basic Auth 账号", http.MethodDelete, "/api/caddy/basic-auth/:index/users/:username"); err != nil {
			respondError(c, http.StatusForbidden, err.Error())
			return
		}
	}
	options := caddy.BasicAuthConfigOptions{Revision: req.Revision, RemoveUsers: req.RemoveUsers}
	if err := app.availability.Load().caddySvc.BasicAuthConfigUpdate(c.Request.Context(), server, idx, req.Realm, req.ForwardHeader, options); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "配置更新成功", nil)
}

// ─── SSL 证书 CRUD ───

func (app *App) caddyCertList(c *gin.Context) {
	result, err := app.availability.Load().caddySvc.CertList(c.Request.Context())
	if err != nil {
		respondCaddyError(c, err)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyCertCreate(c *gin.Context) {
	var req caddy.CertForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.CertCreate(c.Request.Context(), req); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "Caddy SSL 证书创建成功", nil)
}

func (app *App) caddyCertUpdate(c *gin.Context) {
	var req caddy.CertForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().caddySvc.CertUpdate(c.Request.Context(), c.Param("key"), req); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "Caddy SSL 证书更新成功", nil)
}

func (app *App) caddyCertDelete(c *gin.Context) {
	if err := app.availability.Load().caddySvc.CertDelete(c.Request.Context(), c.Param("key")); err != nil {
		respondCaddyError(c, err, http.StatusBadRequest)
		return
	}
	respondSuccess(c, "Caddy SSL 证书删除成功", nil)
}

// ─── 辅助函数 ───

func respondCaddyError(c *gin.Context, err error, fallbackStatus ...int) {
	status := http.StatusInternalServerError
	if len(fallbackStatus) > 0 {
		status = fallbackStatus[0]
	}
	upstreamErr, isUpstream := errors.AsType[*pkgCaddy.HTTPError](err)
	_, isTransport := errors.AsType[*url.Error](err)
	switch {
	case errors.Is(err, caddy.ErrRouteRevisionRequired):
		status = http.StatusPreconditionRequired
	case errors.Is(err, caddy.ErrRouteRevisionChanged):
		status = http.StatusConflict
	case errors.Is(err, caddy.ErrInvalidRoutePosition):
		status = http.StatusBadRequest
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
	case errors.Is(err, caddy.ErrServerNameInvalid), errors.Is(err, caddy.ErrServerConfigInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, caddy.ErrServerNotFound), errors.Is(err, caddy.ErrRouteNotFound):
		status = http.StatusNotFound
	case errors.Is(err, caddy.ErrServerExists), errors.Is(err, caddy.ErrDefaultServer), errors.Is(err, caddy.ErrLastServer):
		status = http.StatusConflict
	case isUpstream:
		if upstreamErr.StatusCode >= http.StatusBadRequest && upstreamErr.StatusCode < http.StatusInternalServerError {
			status = upstreamErr.StatusCode
		} else {
			status = http.StatusBadGateway
		}
	case isTransport:
		status = http.StatusBadGateway
	}
	respondError(c, status, err.Error())
}
