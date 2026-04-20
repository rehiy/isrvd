package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	"isrvd/pkgs/docker"
)

// ListImages 列出镜像
func (h *DockerHandler) ListImages(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"

	result, err := h.service.ListImages(c.Request.Context(), all)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取镜像列表失败")
		return
	}

	helper.RespondSuccess(c, "Images listed successfully", result)
}

// ImageAction 镜像操作
func (h *DockerHandler) ImageAction(c *gin.Context) {
	var req docker.ImageActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	if err := h.service.ImageAction(c.Request.Context(), req.ID, req.Action); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, req.Action+"镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Image "+req.Action+" successfully", nil)
}

// PullImage 拉取镜像
func (h *DockerHandler) PullImage(c *gin.Context) {
	var req docker.ImagePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, imageRef, err := h.service.PullImage(c.Request.Context(), req.Image, req.Tag)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像拉取成功", gin.H{"image": imageRef, "message": msg})
}

// TagImage 镜像打标签
func (h *DockerHandler) TagImage(c *gin.Context) {
	var req docker.ImageTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	if err := h.service.TagImage(c.Request.Context(), req.ID, req.RepoTag); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "镜像打标签失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像打标签成功", nil)
}

// SearchImages 搜索镜像
func (h *DockerHandler) SearchImages(c *gin.Context) {
	term := c.Param("term")
	if term == "" {
		helper.RespondError(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	results, err := h.service.SearchImages(c.Request.Context(), term)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "搜索镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Images searched successfully", results)
}

// BuildImage 构建镜像
func (h *DockerHandler) BuildImage(c *gin.Context) {
	var req docker.ImageBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, err := h.service.BuildImage(c.Request.Context(), req.Dockerfile, req.Tag)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "构建镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像构建成功", gin.H{"tag": req.Tag, "message": msg})
}

// InspectImage 获取镜像详情
func (h *DockerHandler) InspectImage(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "镜像ID不能为空")
		return
	}

	result, err := h.service.InspectImage(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取镜像详情失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Image inspected successfully", result)
}
