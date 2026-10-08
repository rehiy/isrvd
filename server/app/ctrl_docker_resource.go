package app

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/service/docker"
)

func (app *App) dockerNetworkList(c *gin.Context) {
	result, err := app.dockerSvc.NetworkList(c.Request.Context())
	respondResultMsg(c, "获取网络列表成功", result, err)
}

func (app *App) dockerNetworkAction(c *gin.Context) {
	req := docker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.NetworkAction(c.Request.Context(), req)
	respondResultMsg(c, "网络操作成功", nil, err)
}

func (app *App) dockerNetworkCreate(c *gin.Context) {
	var req docker.NetworkSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.NetworkCreate(c.Request.Context(), req)
	respondResultMsg(c, "网络创建成功", result, err)
}

func (app *App) dockerNetworkInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.NetworkInspect(c.Request.Context(), id)
	respondResultMsg(c, "获取网络详情成功", result, err)
}

// ─── 数据卷 ───

func (app *App) dockerVolumeList(c *gin.Context) {
	result, err := app.dockerSvc.VolumeList(c.Request.Context())
	respondResultMsg(c, "获取数据卷列表成功", result, err)
}

func (app *App) dockerVolumeAction(c *gin.Context) {
	req := docker.ActionRequest{
		ID: c.Param("name"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.VolumeAction(c.Request.Context(), req)
	respondResultMsg(c, "数据卷操作成功", nil, err)
}

func (app *App) dockerVolumeCreate(c *gin.Context) {
	var req docker.VolumeSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.VolumeCreate(c.Request.Context(), req)
	respondResultMsg(c, "数据卷创建成功", result, err)
}

func (app *App) dockerVolumeInspect(c *gin.Context) {
	name := c.Param("name")
	result, err := app.dockerSvc.VolumeInspect(c.Request.Context(), name)
	respondResultMsg(c, "获取数据卷详情成功", result, err)
}

// ─── 镜像仓库 ───
