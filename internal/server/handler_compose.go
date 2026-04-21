package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	svcCompose "isrvd/internal/service/compose"
)

func (app *App) composeDeploy(c *gin.Context) {
	var req svcCompose.DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.composeSvc.Deploy(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "部署成功", result)
}

func (app *App) composeRedeploy(c *gin.Context) {
	var req svcCompose.RedeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.composeSvc.RedeployDocker(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "重建成功", result)
}
