package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"isrvd/config"
	svcCompose "isrvd/internal/service/compose"
	pkgCompose "isrvd/pkgs/compose"
)

// defineComposeRoutes 定义 Compose 模块路由
func (app *App) defineComposeRoutes() []Route {
	return []Route{
		// Docker Compose
		{Method: "GET", Path: "/compose/docker/:name", Handler: app.composeDockerInspect, Module: "compose", Label: "读取 Docker Compose 配置"},
		{Method: "POST", Path: "/compose/docker", Handler: app.composeDockerDeploy, Module: "compose", Label: "部署 Docker Compose 应用"},
		{Method: "PUT", Path: "/compose/docker/:name", Handler: app.composeDockerRedeploy, Module: "compose", Label: "重新部署 Docker Compose 应用"},
		// Swarm Compose
		{Method: "GET", Path: "/compose/swarm/:name", Handler: app.composeSwarmInspect, Module: "compose", Label: "读取 Swarm Stack 配置"},
		{Method: "POST", Path: "/compose/swarm", Handler: app.composeSwarmDeploy, Module: "compose", Label: "部署 Swarm Stack 应用"},
		{Method: "PUT", Path: "/compose/swarm/:name", Handler: app.composeSwarmRedeploy, Module: "compose", Label: "重新部署 Swarm Stack 应用"},
	}
}

func (app *App) composeDockerInspect(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}

	content, _, err := app.composeSvc.DockerContent(c.Request.Context(), name)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 compose 文件成功", gin.H{"content": content})
}

func (app *App) composeSwarmInspect(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}

	content, err := app.composeSvc.SwarmContent(c.Request.Context(), name)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 compose 文件成功", gin.H{"content": content})
}

func (app *App) composeDockerDeploy(c *gin.Context) {
	req, ok := bindComposeDeployRequest(c)
	if !ok {
		return
	}
	result, err := app.composeSvc.DockerDeploy(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "部署成功", result)
}

func (app *App) composeSwarmDeploy(c *gin.Context) {
	req, ok := bindComposeDeployRequest(c)
	if !ok {
		return
	}
	result, err := app.composeSvc.SwarmDeploy(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "部署成功", result)
}

func (app *App) composeDockerRedeploy(c *gin.Context) {
	var req svcCompose.RedeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	name, ok := composeNameParam(c)
	if !ok {
		return
	}
	result, err := app.composeSvc.DockerRedeploy(c.Request.Context(), name, req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "重建成功", result)
}

func (app *App) composeSwarmRedeploy(c *gin.Context) {
	var req svcCompose.RedeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	name, ok := composeNameParam(c)
	if !ok {
		return
	}
	result, err := app.composeSvc.SwarmRedeploy(c.Request.Context(), name, req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "重建成功", result)
}

// ─── 辅助函数 ───

func composeNameParam(c *gin.Context) (string, bool) {
	name := c.Param("name")
	if err := pkgCompose.ValidateProjectName(name); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return "", false
	}
	return name, true
}

// bindComposeDeployRequest 解析 JSON 或 multipart form 的部署请求。
func bindComposeDeployRequest(c *gin.Context) (svcCompose.DeployRequest, bool) {
	var req svcCompose.DeployRequest
	if c.Request.ContentLength > config.Server.MaxUploadSize {
		respondError(c, http.StatusBadRequest, "文件大小超过限制")
		return req, false
	}

	if strings.HasPrefix(c.ContentType(), "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return req, false
		}
	} else {
		req.Content = c.PostForm("content")
		req.InitURL = c.PostForm("initURL")
		if fh, err := c.FormFile("initFile"); err == nil {
			if fh.Size > config.Server.MaxUploadSize {
				respondError(c, http.StatusBadRequest, "文件大小超过限制")
				return req, false
			}
			f, err := fh.Open()
			if err != nil {
				respondError(c, http.StatusBadRequest, "读取上传文件失败: "+err.Error())
				return req, false
			}
			req.InitFile = f
		}
	}

	if req.Content == "" {
		respondError(c, http.StatusBadRequest, "content 不能为空")
		return req, false
	}
	return req, true
}
