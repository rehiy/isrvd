package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	svcCompose "isrvd/internal/service/compose"
)

func (app *App) composeGetDockerContent(c *gin.Context) {
	name := c.Param("name")
	content, err := app.composeSvc.GetContent(c.Request.Context(), svcCompose.TargetDocker, name)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "获取 compose 文件成功", gin.H{"content": content})
}

func (app *App) composeGetSwarmContent(c *gin.Context) {
	name := c.Param("name")
	content, err := app.composeSvc.GetContent(c.Request.Context(), svcCompose.TargetSwarm, name)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "获取 compose 文件成功", gin.H{"content": content})
}

func (app *App) composeDeployDocker(c *gin.Context) {
	req := svcCompose.DeployDockerRequest{
		Content:     c.PostForm("content"),
		ProjectName: c.PostForm("projectName"),
		InitURL:     c.PostForm("initURL"),
	}
	if req.Content == "" || req.ProjectName == "" {
		helper.RespondError(c, http.StatusBadRequest, "content 和 projectName 不能为空")
		return
	}

	// 读取上传的 zip 文件（可选），流式传入 service 避免内存双份拷贝
	if fh, err := c.FormFile("initFile"); err == nil {
		f, err := fh.Open()
		if err != nil {
			helper.RespondError(c, http.StatusBadRequest, "读取上传文件失败: "+err.Error())
			return
		}
		req.InitFile = f
		defer f.Close()
	}

	result, err := app.composeSvc.DeployDocker(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "部署成功", result)
}

func (app *App) composeDeploySwarm(c *gin.Context) {
	var req svcCompose.DeploySwarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.composeSvc.DeploySwarm(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "部署成功", result)
}

func (app *App) composeRedeployDocker(c *gin.Context) {
	name := c.Param("name")
	var req svcCompose.RedeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.composeSvc.RedeployDocker(c.Request.Context(), name, req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "重建成功", result)
}

func (app *App) composeRedeploySwarm(c *gin.Context) {
	name := c.Param("name")
	var req svcCompose.RedeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.composeSvc.RedeploySwarm(c.Request.Context(), name, req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "重建成功", result)
}
