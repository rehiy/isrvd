package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	svcCaddy "isrvd/internal/service/caddy"
	pkgCaddy "isrvd/pkgs/caddy"
)

// defineCaddyRoutes 定义 Caddy 模块路由
func (app *App) defineCaddyRoutes() []Route {
	return []Route{
		// 概览与原始配置
		{Method: "GET", Path: "/caddy/info", Handler: app.caddyInfoInspect, Module: "caddy", Label: "查询 Caddy 概览"},
		{Method: "GET", Path: "/caddy/config", Handler: app.caddyConfigInspect, Module: "caddy", Label: "查询 Caddy 完整配置"},
		{Method: "POST", Path: "/caddy/config", Handler: app.caddyConfigLoad, Module: "caddy", Label: "整体替换 Caddy 配置"},
		// 全局选项
		{Method: "GET", Path: "/caddy/global", Handler: app.caddyGlobalInspect, Module: "caddy", Label: "查询 Caddy 全局选项"},
		{Method: "PUT", Path: "/caddy/global", Handler: app.caddyGlobalUpdate, Module: "caddy", Label: "更新 Caddy 全局选项"},
		// 路由 CRUD（默认 server=srv0，可通过 query 指定）
		{Method: "GET", Path: "/caddy/routes", Handler: app.caddyRouteList, Module: "caddy", Label: "查询 Caddy 路由列表"},
		{Method: "GET", Path: "/caddy/route/:index", Handler: app.caddyRouteInspect, Module: "caddy", Label: "获取 Caddy 路由详情"},
		{Method: "POST", Path: "/caddy/route", Handler: app.caddyRouteCreate, Module: "caddy", Label: "创建 Caddy 路由"},
		{Method: "PUT", Path: "/caddy/route/:index", Handler: app.caddyRouteUpdate, Module: "caddy", Label: "更新 Caddy 路由"},
		{Method: "DELETE", Path: "/caddy/route/:index", Handler: app.caddyRouteDelete, Module: "caddy", Label: "删除 Caddy 路由"},
		// TLS 证书 CRUD
		{Method: "GET", Path: "/caddy/certs", Handler: app.caddyCertList, Module: "caddy", Label: "查询 Caddy TLS 证书列表"},
		{Method: "POST", Path: "/caddy/cert", Handler: app.caddyCertCreate, Module: "caddy", Label: "创建 Caddy TLS 证书"},
		{Method: "PUT", Path: "/caddy/cert/:key", Handler: app.caddyCertUpdate, Module: "caddy", Label: "更新 Caddy TLS 证书"},
		{Method: "DELETE", Path: "/caddy/cert/:key", Handler: app.caddyCertDelete, Module: "caddy", Label: "删除 Caddy TLS 证书"},
	}
}

// ─── 概览与原始配置 ───

func (app *App) caddyInfoInspect(c *gin.Context) {
	result, err := app.caddySvc.Info(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyConfigInspect(c *gin.Context) {
	result, err := app.caddySvc.ConfigAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
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
	if err := app.caddySvc.ConfigLoad(c.Request.Context(), req.Config); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "Caddy 配置已加载", nil)
}

// ─── 全局选项 ───

func (app *App) caddyGlobalInspect(c *gin.Context) {
	result, err := app.caddySvc.Global(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyGlobalUpdate(c *gin.Context) {
	var req svcCaddy.GlobalForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.caddySvc.GlobalUpdate(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "全局选项已更新", nil)
}

// ─── 路由 CRUD ───

func (app *App) caddyRouteList(c *gin.Context) {
	server := c.Query("server")
	result, err := app.caddySvc.RouteList(c.Request.Context(), server)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
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
	result, err := app.caddySvc.RouteInspect(c.Request.Context(), server, idx)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyRouteCreate(c *gin.Context) {
	server := c.Query("server")
	var req pkgCaddy.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	idx, err := app.caddySvc.RouteCreate(c.Request.Context(), server, req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
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
	if err := app.caddySvc.RouteUpdate(c.Request.Context(), server, idx, req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
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
	if err := app.caddySvc.RouteDelete(c.Request.Context(), server, idx); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "Caddy 路由删除成功", nil)
}

// ─── TLS 证书 CRUD ───

func (app *App) caddyCertList(c *gin.Context) {
	result, err := app.caddySvc.CertList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) caddyCertCreate(c *gin.Context) {
	var req svcCaddy.CertForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.caddySvc.CertCreate(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "Caddy TLS 证书创建成功", nil)
}

func (app *App) caddyCertUpdate(c *gin.Context) {
	var req svcCaddy.CertForm
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.caddySvc.CertUpdate(c.Request.Context(), c.Param("key"), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "Caddy TLS 证书更新成功", nil)
}

func (app *App) caddyCertDelete(c *gin.Context) {
	if err := app.caddySvc.CertDelete(c.Request.Context(), c.Param("key")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "Caddy TLS 证书删除成功", nil)
}
