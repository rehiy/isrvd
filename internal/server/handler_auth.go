package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	svcSystem "isrvd/internal/service/system"
)

func (app *App) authInfo(c *gin.Context) {
	username := c.GetString("username")
	helper.RespondSuccess(c, "ok", app.authSvc.GetAuthInfo(username))
}

func (app *App) login(c *gin.Context) {
	var req svcSystem.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.authSvc.Login(req)
	if err != nil {
		helper.RespondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	helper.RespondSuccess(c, "Login successful", resp)
}
