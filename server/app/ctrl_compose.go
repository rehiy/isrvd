package app

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkgCompose "isrvd/pkgs/compose"
	"isrvd/server/service/compose"

	"isrvd/server/config"
)

// defineComposeRoutes 定义 Compose 模块路由
func (app *App) defineComposeRoutes() []Route {
	return []Route{
		// Docker Compose
		{Method: "GET", Path: "/compose/docker/:name", Handler: app.composeDockerInspect, Module: "compose", Label: "读取 Docker Compose 配置"},
		{Method: "GET", Path: "/compose/docker/:name/history", Handler: app.composeDockerHistoryList, Module: "compose", Label: "读取 Docker Compose 部署记录"},
		{Method: "POST", Path: "/compose/docker", Handler: app.composeDockerDeploy, Module: "compose", Label: "部署 Docker Compose 应用"},
		{Method: "PUT", Path: "/compose/docker/:name", Handler: app.composeDockerRedeploy, Module: "compose", Label: "重新部署 Docker Compose 应用"},
		// Swarm Compose
		{Method: "GET", Path: "/compose/swarm/:name", Handler: app.composeSwarmInspect, Module: "compose", Label: "读取 Swarm Stack 配置"},
		{Method: "GET", Path: "/compose/swarm/:name/history", Handler: app.composeSwarmHistoryList, Module: "compose", Label: "读取 Swarm Stack 部署记录"},
		{Method: "POST", Path: "/compose/swarm", Handler: app.composeSwarmDeploy, Module: "compose", Label: "部署 Swarm Stack 应用"},
		{Method: "PUT", Path: "/compose/swarm/:name", Handler: app.composeSwarmRedeploy, Module: "compose", Label: "重新部署 Swarm Stack 应用"},
	}
}

func (app *App) composeDockerInspect(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}

	forceRuntime := c.Query("force") == "true"
	if id := c.Query("revision"); id != "" {
		detail, err := app.composeSvc.HistoryInspect(c.Request.Context(), "docker", name, id)
		respondResult(c, detail, err)
		return
	}
	detail, err := app.composeSvc.DockerInspect(c.Request.Context(), name, forceRuntime)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 compose 文件成功", detail)
}

func (app *App) composeSwarmInspect(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}

	forceRuntime := c.Query("force") == "true"
	if id := c.Query("revision"); id != "" {
		detail, err := app.composeSvc.HistoryInspect(c.Request.Context(), "swarm", name, id)
		respondResult(c, detail, err)
		return
	}
	detail, err := app.composeSvc.SwarmInspect(c.Request.Context(), name, forceRuntime)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 compose 文件成功", detail)
}

func (app *App) composeDockerHistoryList(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}
	records, err := app.composeSvc.HistoryList(c.Request.Context(), "docker", name)
	respondResult(c, records, err)
}

func (app *App) composeSwarmHistoryList(c *gin.Context) {
	name, ok := composeNameParam(c)
	if !ok {
		return
	}
	records, err := app.composeSvc.HistoryList(c.Request.Context(), "swarm", name)
	respondResult(c, records, err)
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
	var req compose.RedeployRequest
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
	var req compose.RedeployRequest
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
func bindComposeDeployRequest(c *gin.Context) (compose.DeployRequest, bool) {
	var req compose.DeployRequest
	if c.Request.ContentLength > config.Current().Server.MaxUploadSize {
		respondError(c, http.StatusBadRequest, "文件大小超过限制")
		return req, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Current().Server.MaxUploadSize)

	if strings.HasPrefix(c.ContentType(), "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return req, false
		}
	} else {
		req.Content = c.PostForm("content")
		if v, ok := c.GetPostForm("envContent"); ok {
			req.EnvContent = &v
		}
		req.InitURL = c.PostForm("initURL")
		if fh, err := c.FormFile("initFile"); err == nil {
			if fh.Size > config.Current().Server.MaxUploadSize {
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
