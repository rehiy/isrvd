package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	dockerPkg "isrvd/pkgs/docker"
	"isrvd/server/helper"
)

// ListRegistries 列出已配置的镜像仓库
func (h *DockerHandler) ListRegistries(c *gin.Context) {
	result := h.service.ListRegistries()
	helper.RespondSuccess(c, "Registries listed successfully", result)
}

// PushImage 推送镜像到仓库
func (h *DockerHandler) PushImage(c *gin.Context) {
	var req dockerPkg.ImagePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, targetRef, err := h.service.PushImage(c.Request.Context(), req)
	if err != nil {
		logman.Error("Push image failed", "image", targetRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "推送镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像推送成功", gin.H{"image": req.Image, "target": targetRef, "message": msg})
}

// PullFromRegistry 从仓库拉取镜像
func (h *DockerHandler) PullFromRegistry(c *gin.Context) {
	var req dockerPkg.ImagePullFromRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, imageRef, err := h.service.PullFromRegistry(c.Request.Context(), req)
	if err != nil {
		logman.Error("Pull from registry failed", "image", imageRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "从仓库拉取镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像拉取成功", gin.H{"image": imageRef, "message": msg})
}
