package app

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/service/docker"
)

func (app *App) dockerImageList(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"
	result, err := app.dockerSvc.ImageList(c.Request.Context(), all)
	respondResultMsg(c, "获取镜像列表成功", result, err)
}

func (app *App) dockerImageAction(c *gin.Context) {
	req := docker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ImageAction(c.Request.Context(), req)
	respondResultMsg(c, "镜像操作成功", nil, err)
}

func (app *App) dockerImageTag(c *gin.Context) {
	req := docker.ImageTagRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ImageTag(c.Request.Context(), req)
	respondResultMsg(c, "添加镜像标签成功", nil, err)
}

func (app *App) dockerImageSearch(c *gin.Context) {
	name := c.Query("name")
	result, err := app.dockerSvc.ImageSearch(c.Request.Context(), name)
	respondResultMsg(c, "搜索镜像成功", result, err)
}

func (app *App) dockerImageBuild(c *gin.Context) {
	var req docker.ImageBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImageBuild(c.Request.Context(), req)
	respondResultMsg(c, "镜像构建成功", result, err)
}

func (app *App) dockerImagePrune(c *gin.Context) {
	var req docker.ImagePruneRequest
	// 请求体可选；空 JSON 表示仅清理悬空层
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePrune(c.Request.Context(), req)
	respondResultMsg(c, "镜像清理成功", result, err)
}

func (app *App) dockerImageInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ImageInspect(c.Request.Context(), id)
	respondResultMsg(c, "获取镜像详情成功", result, err)
}

// ─── 网络 ───

func (app *App) dockerImagePush(c *gin.Context) {
	var req docker.ImagePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePush(c.Request.Context(), req)
	respondResultMsg(c, "镜像推送成功", result, err)
}

func (app *App) dockerImagePull(c *gin.Context) {
	var req docker.ImagePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePull(c.Request.Context(), req)
	respondResultMsg(c, "镜像拉取成功", result, err)
}

// ─── 容器文件管理 ───
