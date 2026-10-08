package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/service/docker"
)

func (app *App) dockerContainerList(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"
	result, err := app.dockerSvc.ContainerList(c.Request.Context(), all, c.Query("filters"))
	if err != nil {
		// Docker filters 解析失败属于调用方错误（SDK 以 InvalidParameter 标记），返回 400
		var invalid interface{ InvalidParameter() }
		if errors.As(err, &invalid) {
			respondError(c, http.StatusBadRequest, "filters 参数格式错误: "+err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取容器列表成功", result)
}

func (app *App) dockerContainerCreate(c *gin.Context) {
	var req docker.ContainerSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ContainerCreate(c.Request.Context(), req)
	respondResultMsg(c, "容器创建成功", result, err)
}

func (app *App) dockerContainerInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ContainerInspect(c.Request.Context(), id)
	respondResultMsg(c, "获取容器详情成功", result, err)
}

func (app *App) dockerContainerStats(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ContainerStats(c.Request.Context(), id)
	respondResultMsg(c, "获取容器资源统计成功", result, err)
}

func (app *App) dockerContainerAction(c *gin.Context) {
	req := docker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ContainerAction(c.Request.Context(), req)
	respondResultMsg(c, "容器操作成功", nil, err)
}

func (app *App) dockerContainerLogs(c *gin.Context) {
	req := docker.ContainerLogsRequest{
		ID:   c.Param("id"),
		Tail: c.DefaultQuery("tail", "100"),
	}
	if req.ID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}
	result, err := app.dockerSvc.ContainerLogs(c.Request.Context(), req)
	respondResultMsg(c, "获取容器日志成功", result, err)
}

func (app *App) dockerContainerLogsStream(c *gin.Context) {
	req := docker.ContainerLogsRequest{
		ID:   c.Param("id"),
		Tail: c.DefaultQuery("tail", "100"),
	}
	if req.ID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}
	w, err := httpd.NewEventWriter(c.Writer)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	app.dockerSvc.ContainerLogsStream(c.Request.Context(), w, req)
}

func (app *App) dockerContainerExec(c *gin.Context) {
	containerID := c.Param("id")
	shell := c.DefaultQuery("shell", "/bin/sh")
	if containerID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}

	app.serveWebSocket(c, func(conn *websocket.ServerConn) {
		app.dockerSvc.ContainerExec(c.Request.Context(), conn, containerID, shell)
	})
}

// ─── 镜像 ───
