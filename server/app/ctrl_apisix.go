package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	pkgApisix "isrvd/pkgs/apisix"
	"isrvd/server/service/apisix"
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
		// 指标开关（抓取出口见 /metrics/apisix）
		{Method: "PUT", Path: "/apisix/metrics", Handler: app.apisixMetricsUpdate, Module: "apisix", Label: "开关 APISIX 指标"},
		// Consumer 管理
		{Method: "GET", Path: "/apisix/consumers", Handler: app.apisixConsumerList, Module: "apisix", Label: "查询 APISIX 消费者列表"},
		{Method: "GET", Path: "/apisix/consumer/:username/key", Handler: app.apisixConsumerKeyInspect, Module: "apisix", Label: "读取 APISIX 消费者完整 Token", ExplicitGrant: true, Audit: AuditAlways},
		{Method: "PUT", Path: "/apisix/consumer/:username/key", Handler: app.apisixConsumerKeyUpdate, Module: "apisix", Label: "签发 APISIX 消费者 Token", ExplicitGrant: true, ExtraPerm: "GET /api/apisix/consumer/:username/key"},
		{Method: "POST", Path: "/apisix/consumer", Handler: app.apisixConsumerCreate, Module: "apisix", Label: "创建 APISIX 消费者"},
		{Method: "PUT", Path: "/apisix/consumer/:username", Handler: app.apisixConsumerUpdate, Module: "apisix", Label: "更新 APISIX 消费者"},
		{Method: "DELETE", Path: "/apisix/consumer/:username", Handler: app.apisixConsumerDelete, Module: "apisix", Label: "删除 APISIX 消费者"},
		// 访问授权
		{Method: "GET", Path: "/apisix/whitelist", Handler: app.apisixWhitelistInspect, Module: "apisix", Label: "查询 APISIX 访问授权"},
		{Method: "POST", Path: "/apisix/whitelist", Handler: app.apisixWhitelistCreate, Module: "apisix", Label: "配置 APISIX 路由访问授权"},
		{Method: "POST", Path: "/apisix/whitelist/user", Handler: app.apisixWhitelistUserCreate, Module: "apisix", Label: "新建用户并加入 APISIX 访问授权"},
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

func (app *App) apisixMetricsUpdate(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled" binding:"required"` // true 开启，false 关闭
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondMetricsToggled(c, *req.Enabled, app.availability.Load().apisixSvc.MetricsUpdate(c.Request.Context(), *req.Enabled), apisix.ErrMetricsExternallyManaged)
}

func (app *App) apisixRouteList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.RouteList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixRouteInspect(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.RouteInspect(c.Request.Context(), c.Param("id"))
	respondResult(c, result, err)
}

func (app *App) apisixRouteCreate(c *gin.Context) {
	var req pkgApisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.RouteCreate(c.Request.Context(), req)
	respondResultMsg(c, "路由创建成功", result, err)
}

func (app *App) apisixRouteUpdate(c *gin.Context) {
	var req pkgApisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.RouteUpdate(c.Request.Context(), c.Param("id"), req)
	respondResultMsg(c, "路由更新成功", result, err)
}

func (app *App) apisixRouteStatusPatch(c *gin.Context) {
	var req struct {
		Status int `json:"status"` // 路由状态：1 启用，0 禁用
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.availability.Load().apisixSvc.RouteStatusPatch(c.Request.Context(), c.Param("id"), req.Status)
	respondResultMsg(c, "路由状态更新成功", nil, err)
}

func (app *App) apisixRouteDelete(c *gin.Context) {
	err := app.availability.Load().apisixSvc.RouteDelete(c.Request.Context(), c.Param("id"))
	respondResultMsg(c, "路由删除成功", nil, err)
}

func (app *App) apisixConsumerList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.ConsumerList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixConsumerCreate(c *gin.Context) {
	var req struct {
		Username string         `json:"username" binding:"required"` // 消费者用户名
		Desc     string         `json:"desc"`                        // 描述
		Plugins  map[string]any `json:"plugins"`                     // 插件配置
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.ConsumerCreate(c.Request.Context(), req.Username, req.Desc, req.Plugins)
	respondResultMsg(c, "消费者创建成功", result, err)
}

func (app *App) apisixConsumerKeyInspect(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	result, err := app.availability.Load().apisixSvc.ConsumerKeyInspect(c.Request.Context(), c.Param("username"))
	if err != nil {
		status, msg := consumerKeyError(err)
		respondError(c, status, msg)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixConsumerKeyUpdate(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	if err != nil || !json.Valid(raw) {
		respondConsumerKeyBindError(c, err)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	var req apisix.ConsumerKeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondConsumerKeyBindError(c, err)
		return
	}
	result, err := app.availability.Load().apisixSvc.ConsumerKeyUpdate(c.Request.Context(), c.Param("username"), req)
	if err != nil {
		status, msg := consumerKeyError(err)
		respondError(c, status, msg)
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) apisixConsumerUpdate(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		Desc    string         `json:"desc"`    // 描述
		Plugins map[string]any `json:"plugins"` // 插件配置
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.availability.Load().apisixSvc.ConsumerUpdate(c.Request.Context(), username, req.Desc, req.Plugins); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "消费者更新成功", gin.H{"username": username, "desc": req.Desc})
}

func (app *App) apisixConsumerDelete(c *gin.Context) {
	err := app.availability.Load().apisixSvc.ConsumerDelete(c.Request.Context(), c.Param("username"))
	respondResultMsg(c, "消费者删除成功", nil, err)
}

func (app *App) apisixWhitelistInspect(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.WhitelistList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixWhitelistCreate(c *gin.Context) {
	var req apisix.WhitelistCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.WhitelistCreate(c.Request.Context(), req)
	respondResultMsg(c, "访问授权配置成功", result, err)
}

func (app *App) apisixWhitelistUserCreate(c *gin.Context) {
	var req apisix.WhitelistUserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.WhitelistUserCreate(c.Request.Context(), req)
	respondResultMsg(c, "用户创建并加入访问授权成功", result, err)
}

func (app *App) apisixPluginConfigList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.PluginConfigList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixPluginConfigInspect(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.PluginConfigInspect(c.Request.Context(), c.Param("id"))
	respondResult(c, result, err)
}

func (app *App) apisixPluginConfigCreate(c *gin.Context) {
	var req pkgApisix.PluginConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.PluginConfigCreate(c.Request.Context(), req)
	respondResultMsg(c, "插件配置创建成功", result, err)
}

func (app *App) apisixPluginConfigUpdate(c *gin.Context) {
	var req pkgApisix.PluginConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.PluginConfigUpdate(c.Request.Context(), c.Param("id"), req)
	respondResultMsg(c, "插件配置更新成功", result, err)
}

func (app *App) apisixPluginConfigDelete(c *gin.Context) {
	err := app.availability.Load().apisixSvc.PluginConfigDelete(c.Request.Context(), c.Param("id"))
	respondResultMsg(c, "插件配置删除成功", nil, err)
}

func (app *App) apisixUpstreamList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.UpstreamList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixUpstreamInspect(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.UpstreamInspect(c.Request.Context(), c.Param("id"))
	respondResult(c, result, err)
}

func (app *App) apisixUpstreamCreate(c *gin.Context) {
	var req pkgApisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.UpstreamCreate(c.Request.Context(), req)
	respondResultMsg(c, "上游创建成功", result, err)
}

func (app *App) apisixUpstreamUpdate(c *gin.Context) {
	var req pkgApisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.UpstreamUpdate(c.Request.Context(), c.Param("id"), req)
	respondResultMsg(c, "上游更新成功", result, err)
}

func (app *App) apisixUpstreamDelete(c *gin.Context) {
	err := app.availability.Load().apisixSvc.UpstreamDelete(c.Request.Context(), c.Param("id"))
	respondResultMsg(c, "上游删除成功", nil, err)
}

func (app *App) apisixSSLList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.SSLList(c.Request.Context())
	respondResult(c, result, err)
}

func (app *App) apisixSSLInspect(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.SSLInspect(c.Request.Context(), c.Param("id"))
	respondResult(c, result, err)
}

func (app *App) apisixSSLCreate(c *gin.Context) {
	var req pkgApisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.SSLCreate(c.Request.Context(), req)
	respondResultMsg(c, "证书创建成功", result, err)
}

func (app *App) apisixSSLUpdate(c *gin.Context) {
	var req pkgApisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.availability.Load().apisixSvc.SSLUpdate(c.Request.Context(), c.Param("id"), req)
	respondResultMsg(c, "证书更新成功", result, err)
}

func (app *App) apisixSSLDelete(c *gin.Context) {
	err := app.availability.Load().apisixSvc.SSLDelete(c.Request.Context(), c.Param("id"))
	respondResultMsg(c, "证书删除成功", nil, err)
}

func (app *App) apisixPluginList(c *gin.Context) {
	result, err := app.availability.Load().apisixSvc.PluginList(c.Request.Context())
	respondResult(c, result, err)
}

// ─── 辅助函数 ───

func respondConsumerKeyBindError(c *gin.Context, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		respondError(c, http.StatusRequestEntityTooLarge, "请求体过大")
		return
	}
	respondError(c, http.StatusBadRequest, "请求体格式错误")
}

func consumerKeyError(err error) (int, string) {
	switch {
	case errors.Is(err, pkgApisix.ErrConsumerNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, pkgApisix.ErrConsumerKeyMissing):
		return http.StatusConflict, err.Error()
	case errors.Is(err, pkgApisix.ErrConsumerKeyInvalid):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, pkgApisix.ErrConsumerResponseInvalid):
		return http.StatusBadGateway, err.Error()
	}
	if apiErr, ok := errors.AsType[*pkgApisix.APIError](err); ok {
		return http.StatusBadGateway, apiErr.Error()
	}
	return http.StatusBadGateway, "请求 Apisix 失败"
}
