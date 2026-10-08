package app

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/service/docker"
)

// dockerRegistryUpsertRequest 是 service/docker 中 RegistryUpsertRequest 的本地别名
type dockerRegistryUpsertRequest = docker.RegistryUpsertRequest

func (app *App) dockerRegistryList(c *gin.Context) {
	respondSuccess(c, "获取镜像仓库列表成功", app.dockerSvc.RegistryList())
}

func (app *App) dockerRegistryCreate(c *gin.Context) {
	var req dockerRegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.RegistryCreate(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库添加成功", nil)
}

func (app *App) dockerRegistryUpdate(c *gin.Context) {
	originalURL := c.Query("url")
	if originalURL == "" {
		respondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	var req dockerRegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.RegistryUpdate(originalURL, req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库更新成功", nil)
}

func (app *App) dockerRegistryDelete(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		respondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	if err := app.dockerSvc.RegistryDelete(url); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库删除成功", nil)
}
