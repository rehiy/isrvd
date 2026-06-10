package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apisixsvc "isrvd/internal/service/apisix"
	pkgapisix "isrvd/pkgs/apisix"
)

// defineApisixRoutes 定义 Apisix 模块路由
func (app *App) defineApisixRoutes() []Route {
	return []Route{
		// Route 管理
		{Method: "GET", Path: "/apisix/routes", Handler: app.apisixRouteList, Module: "apisix", Label: "查询 APISIX 路由列表"},
		{Method: "GET", Path: "/apisix/route/:id", Handler: app.apisixRouteInspect, Module: "apisix", Label: "获取 APISIX 路由详情"},
		{Method: "POST", Path: "/apisix/route", Handler: app.apisixRouteCreate, Module: "apisix", Label: "创建 APISIX 路由"},
		{Method: "PUT", Path: "/apisix/route/:id", Handler: app.apisixRouteUpdate, Module: "apisix", Label: "更新 APISIX 路由"},
		{Method: "PATCH", Path: "/apisix/route/:id/status", Handler: app.apisixRouteStatusPatch, Module: "apisix", Label: "切换 APISIX 路由状态"},
		{Method: "DELETE", Path: "/apisix/route/:id", Handler: app.apisixRouteDelete, Module: "apisix", Label: "删除 APISIX 路由"},
		// Consumer 管理
		{Method: "GET", Path: "/apisix/consumers", Handler: app.apisixConsumerList, Module: "apisix", Label: "查询 APISIX 消费者列表"},
		{Method: "POST", Path: "/apisix/consumer", Handler: app.apisixConsumerCreate, Module: "apisix", Label: "创建 APISIX 消费者"},
		{Method: "PUT", Path: "/apisix/consumer/:username", Handler: app.apisixConsumerUpdate, Module: "apisix", Label: "更新 APISIX 消费者"},
		{Method: "DELETE", Path: "/apisix/consumer/:username", Handler: app.apisixConsumerDelete, Module: "apisix", Label: "删除 APISIX 消费者"},
		// 白名单
		{Method: "GET", Path: "/apisix/whitelist", Handler: app.apisixWhitelistList, Module: "apisix", Label: "查询 APISIX 白名单"},
		{Method: "POST", Path: "/apisix/whitelist", Handler: app.apisixWhitelistCreate, Module: "apisix", Label: "配置 APISIX 路由白名单"},
		{Method: "POST", Path: "/apisix/whitelist/user", Handler: app.apisixWhitelistUserCreate, Module: "apisix", Label: "新建用户并加入 APISIX 白名单"},
		{Method: "DELETE", Path: "/apisix/whitelist/user", Handler: app.apisixWhitelistRevoke, Module: "apisix", Label: "撤销 APISIX 白名单授权"},
		// PluginConfig 管理
		{Method: "GET", Path: "/apisix/plugin-configs", Handler: app.apisixPluginConfigList, Module: "apisix", Label: "查询 APISIX 插件配置列表"},
		{Method: "GET", Path: "/apisix/plugin-config/:id", Handler: app.apisixPluginConfigInspect, Module: "apisix", Label: "获取 APISIX 插件配置详情"},
		{Method: "POST", Path: "/apisix/plugin-config", Handler: app.apisixPluginConfigCreate, Module: "apisix", Label: "创建 APISIX 插件配置"},
		{Method: "PUT", Path: "/apisix/plugin-config/:id", Handler: app.apisixPluginConfigUpdate, Module: "apisix", Label: "更新 APISIX 插件配置"},
		{Method: "DELETE", Path: "/apisix/plugin-config/:id", Handler: app.apisixPluginConfigDelete, Module: "apisix", Label: "删除 APISIX 插件配置"},
		// Upstream 管理
		{Method: "GET", Path: "/apisix/upstreams", Handler: app.apisixUpstreamList, Module: "apisix", Label: "查询 APISIX 上游列表"},
		{Method: "GET", Path: "/apisix/upstream/:id", Handler: app.apisixUpstreamInspect, Module: "apisix", Label: "获取 APISIX 上游详情"},
		{Method: "POST", Path: "/apisix/upstream", Handler: app.apisixUpstreamCreate, Module: "apisix", Label: "创建 APISIX 上游"},
		{Method: "PUT", Path: "/apisix/upstream/:id", Handler: app.apisixUpstreamUpdate, Module: "apisix", Label: "更新 APISIX 上游"},
		{Method: "DELETE", Path: "/apisix/upstream/:id", Handler: app.apisixUpstreamDelete, Module: "apisix", Label: "删除 APISIX 上游"},
		// SSL 管理
		{Method: "GET", Path: "/apisix/ssls", Handler: app.apisixSSLList, Module: "apisix", Label: "查询 APISIX 证书列表"},
		{Method: "GET", Path: "/apisix/ssl/:id", Handler: app.apisixSSLInspect, Module: "apisix", Label: "获取 APISIX 证书详情"},
		{Method: "POST", Path: "/apisix/ssl", Handler: app.apisixSSLCreate, Module: "apisix", Label: "创建 APISIX 证书"},
		{Method: "PUT", Path: "/apisix/ssl/:id", Handler: app.apisixSSLUpdate, Module: "apisix", Label: "更新 APISIX 证书"},
		{Method: "DELETE", Path: "/apisix/ssl/:id", Handler: app.apisixSSLDelete, Module: "apisix", Label: "删除 APISIX 证书"},
		// 插件列表
		{Method: "GET", Path: "/apisix/plugins", Handler: app.apisixPluginList, Module: "apisix", Label: "查询 APISIX 插件列表"},
	}
}

func (app *App) apisixRouteList(c *gin.Context) {
	result, err := app.apisixSvc.RouteList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixRouteInspect(c *gin.Context) {
	result, err := app.apisixSvc.RouteInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixRouteCreate(c *gin.Context) {
	var req pkgapisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.RouteCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "路由创建成功", result)
}

func (app *App) apisixRouteUpdate(c *gin.Context) {
	var req pkgapisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.RouteUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "路由更新成功", result)
}

func (app *App) apisixRouteStatusPatch(c *gin.Context) {
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.apisixSvc.RouteStatusPatch(c.Request.Context(), c.Param("id"), req.Status); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "路由状态更新成功", nil)
}

func (app *App) apisixRouteDelete(c *gin.Context) {
	if err := app.apisixSvc.RouteDelete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "路由删除成功", nil)
}

func (app *App) apisixConsumerList(c *gin.Context) {
	result, err := app.apisixSvc.ConsumerList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixConsumerCreate(c *gin.Context) {
	var req struct {
		Username string         `json:"username" binding:"required"`
		Desc     string         `json:"desc"`
		Plugins  map[string]any `json:"plugins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.ConsumerCreate(c.Request.Context(), req.Username, req.Desc, req.Plugins)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "消费者创建成功", result)
}

func (app *App) apisixConsumerUpdate(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		Desc    string         `json:"desc"`
		Plugins map[string]any `json:"plugins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.apisixSvc.ConsumerUpdate(c.Request.Context(), username, req.Desc, req.Plugins); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "消费者更新成功", gin.H{"username": username, "desc": req.Desc})
}

func (app *App) apisixConsumerDelete(c *gin.Context) {
	if err := app.apisixSvc.ConsumerDelete(c.Request.Context(), c.Param("username")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "消费者删除成功", nil)
}

func (app *App) apisixWhitelistList(c *gin.Context) {
	result, err := app.apisixSvc.WhitelistList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixWhitelistCreate(c *gin.Context) {
	var req apisixsvc.WhitelistRouteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.WhitelistRouteCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "白名单配置成功", result)
}

func (app *App) apisixWhitelistUserCreate(c *gin.Context) {
	var req apisixsvc.WhitelistUserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.WhitelistUserCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "用户创建并加入白名单成功", result)
}

func (app *App) apisixWhitelistRevoke(c *gin.Context) {
	routeID := c.Query("route_id")
	consumerName := c.Query("consumer_name")
	if routeID == "" || consumerName == "" {
		respondError(c, http.StatusBadRequest, "route_id 和 consumer_name 不能为空")
		return
	}
	if err := app.apisixSvc.WhitelistRevoke(c.Request.Context(), routeID, consumerName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "白名单授权撤销成功", nil)
}

func (app *App) apisixPluginConfigList(c *gin.Context) {
	result, err := app.apisixSvc.PluginConfigList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixPluginConfigInspect(c *gin.Context) {
	result, err := app.apisixSvc.PluginConfigInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixPluginConfigCreate(c *gin.Context) {
	var req pkgapisix.PluginConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.PluginConfigCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "插件配置创建成功", result)
}

func (app *App) apisixPluginConfigUpdate(c *gin.Context) {
	var req pkgapisix.PluginConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.PluginConfigUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "插件配置更新成功", result)
}

func (app *App) apisixPluginConfigDelete(c *gin.Context) {
	if err := app.apisixSvc.PluginConfigDelete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "插件配置删除成功", nil)
}

func (app *App) apisixUpstreamList(c *gin.Context) {
	result, err := app.apisixSvc.UpstreamList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixUpstreamInspect(c *gin.Context) {
	result, err := app.apisixSvc.UpstreamInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixUpstreamCreate(c *gin.Context) {
	var req pkgapisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.UpstreamCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "上游创建成功", result)
}

func (app *App) apisixUpstreamUpdate(c *gin.Context) {
	var req pkgapisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.UpstreamUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "上游更新成功", result)
}

func (app *App) apisixUpstreamDelete(c *gin.Context) {
	if err := app.apisixSvc.UpstreamDelete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "上游删除成功", nil)
}

func (app *App) apisixSSLList(c *gin.Context) {
	result, err := app.apisixSvc.SSLList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixSSLInspect(c *gin.Context) {
	result, err := app.apisixSvc.SSLInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixSSLCreate(c *gin.Context) {
	var req pkgapisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.SSLCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "证书创建成功", result)
}

func (app *App) apisixSSLUpdate(c *gin.Context) {
	var req pkgapisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.SSLUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "证书更新成功", result)
}

func (app *App) apisixSSLDelete(c *gin.Context) {
	if err := app.apisixSvc.SSLDelete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "证书删除成功", nil)
}

func (app *App) apisixPluginList(c *gin.Context) {
	result, err := app.apisixSvc.PluginList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}
