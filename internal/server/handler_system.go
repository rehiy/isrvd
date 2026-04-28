package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/config"
	"isrvd/internal/helper"
	svcSystem "isrvd/internal/service/system"
)

func (app *App) systemStat(c *gin.Context) {
	helper.RespondSuccess(c, "ok", app.systemSvc.Stat(c.Request.Context()))
}

func (app *App) systemProbe(c *gin.Context) {
	helper.RespondSuccess(c, "ok", app.systemSvc.Probe(c.Request.Context()))
}

func (app *App) systemGetSettings(c *gin.Context) {
	if c.Query("reload") == "true" {
		if err := config.Load(); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	helper.RespondSuccess(c, "ok", app.settingsSvc.GetAll())
}

func (app *App) systemUpdateSettings(c *gin.Context) {
	var req svcSystem.UpdateAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.settingsSvc.UpdateAll(req); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "全部配置已保存，部分项需重启生效", nil)
}

func (app *App) systemListMembers(c *gin.Context) {
	helper.RespondSuccess(c, "ok", app.memberSvc.ListMembers())
}

func (app *App) systemCreateMember(c *gin.Context) {
	var req svcSystem.MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.memberSvc.CreateMember(req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.RespondSuccess(c, "成员添加成功", nil)
}

func (app *App) systemUpdateMember(c *gin.Context) {
	username := c.Param("username")
	var req svcSystem.MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.memberSvc.UpdateMember(username, req); err != nil {
		if errors.Is(err, svcSystem.ErrMemberNotFound) {
			helper.RespondError(c, http.StatusNotFound, err.Error())
		} else {
			helper.RespondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	helper.RespondSuccess(c, "成员更新成功", nil)
}

func (app *App) systemDeleteMember(c *gin.Context) {
	username := c.Param("username")
	if err := app.memberSvc.DeleteMember(username); err != nil {
		switch {
		case errors.Is(err, svcSystem.ErrMemberNotFound):
			helper.RespondError(c, http.StatusNotFound, err.Error())
		default:
			helper.RespondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	helper.RespondSuccess(c, "成员删除成功", nil)
}
