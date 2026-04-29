package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	pkgapisix "isrvd/pkgs/apisix"
)

func (app *App) apisixListRoutes(c *gin.Context) {
	result, err := app.apisixSvc.ListRoutes()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixGetRoute(c *gin.Context) {
	result, err := app.apisixSvc.GetRoute(c.Param("id"))
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixCreateRoute(c *gin.Context) {
	var req pkgapisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.CreateRoute(req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route created successfully", result)
}

func (app *App) apisixUpdateRoute(c *gin.Context) {
	var req pkgapisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.UpdateRoute(c.Param("id"), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route updated successfully", result)
}

func (app *App) apisixPatchRouteStatus(c *gin.Context) {
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.apisixSvc.PatchRouteStatus(c.Param("id"), req.Status); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route status updated successfully", nil)
}

func (app *App) apisixDeleteRoute(c *gin.Context) {
	if err := app.apisixSvc.DeleteRoute(c.Param("id")); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route deleted successfully", nil)
}

func (app *App) apisixListConsumers(c *gin.Context) {
	result, err := app.apisixSvc.ListConsumers()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixCreateConsumer(c *gin.Context) {
	var req struct {
		Username string         `json:"username" binding:"required"`
		Desc     string         `json:"desc"`
		Plugins  map[string]any `json:"plugins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.CreateConsumer(req.Username, req.Desc, req.Plugins)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer created successfully", result)
}

func (app *App) apisixUpdateConsumer(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		Desc    string         `json:"desc"`
		Plugins map[string]any `json:"plugins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.apisixSvc.UpdateConsumer(username, req.Desc, req.Plugins); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer updated successfully", gin.H{"username": username, "desc": req.Desc})
}

func (app *App) apisixDeleteConsumer(c *gin.Context) {
	if err := app.apisixSvc.DeleteConsumer(c.Param("username")); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer deleted successfully", nil)
}

func (app *App) apisixGetWhitelist(c *gin.Context) {
	result, err := app.apisixSvc.GetWhitelist()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixRevokeWhitelist(c *gin.Context) {
	var req struct {
		RouteID      string `json:"route_id" binding:"required"`
		ConsumerName string `json:"consumer_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.apisixSvc.RevokeWhitelist(req.RouteID, req.ConsumerName); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Whitelist revoked successfully", nil)
}

func (app *App) apisixListPluginConfigs(c *gin.Context) {
	result, err := app.apisixSvc.ListPluginConfigs()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixListUpstreams(c *gin.Context) {
	result, err := app.apisixSvc.ListUpstreams()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixGetUpstream(c *gin.Context) {
	result, err := app.apisixSvc.GetUpstream(c.Param("id"))
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixCreateUpstream(c *gin.Context) {
	var req pkgapisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.CreateUpstream(req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Upstream created successfully", result)
}

func (app *App) apisixUpdateUpstream(c *gin.Context) {
	var req pkgapisix.Upstream
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.UpdateUpstream(c.Param("id"), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Upstream updated successfully", result)
}

func (app *App) apisixDeleteUpstream(c *gin.Context) {
	if err := app.apisixSvc.DeleteUpstream(c.Param("id")); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Upstream deleted successfully", nil)
}

func (app *App) apisixListSSLs(c *gin.Context) {
	result, err := app.apisixSvc.ListSSLs()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixGetSSL(c *gin.Context) {
	result, err := app.apisixSvc.GetSSL(c.Param("id"))
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

func (app *App) apisixCreateSSL(c *gin.Context) {
	var req pkgapisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.CreateSSL(req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "SSL created successfully", result)
}

func (app *App) apisixUpdateSSL(c *gin.Context) {
	var req pkgapisix.SSL
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.apisixSvc.UpdateSSL(c.Param("id"), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "SSL updated successfully", result)
}

func (app *App) apisixDeleteSSL(c *gin.Context) {
	if err := app.apisixSvc.DeleteSSL(c.Param("id")); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "SSL deleted successfully", nil)
}

func (app *App) apisixListPlugins(c *gin.Context) {
	result, err := app.apisixSvc.ListPlugins()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}
