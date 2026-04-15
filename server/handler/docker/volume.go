package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/pkgs/docker"
	"isrvd/server/helper"
)

// ListVolumes 列出卷
func (h *DockerHandler) ListVolumes(c *gin.Context) {
	result, err := h.service.ListVolumes(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取卷列表失败")
		return
	}

	helper.RespondSuccess(c, "Volumes listed successfully", result)
}

// VolumeAction 卷操作
func (h *DockerHandler) VolumeAction(c *gin.Context) {
	var req docker.VolumeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	if err := h.service.VolumeAction(c.Request.Context(), req.Name, req.Action); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, req.Action+"卷失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Volume "+req.Action+" successfully", nil)
}

// CreateVolume 创建卷
func (h *DockerHandler) CreateVolume(c *gin.Context) {
	var req docker.VolumeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	name, mountpoint, err := h.service.CreateVolume(c.Request.Context(), req.Name, req.Driver)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "创建卷失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "卷创建成功", gin.H{"name": name, "mountpoint": mountpoint})
}

// VolumeInspect 获取卷详情
func (h *DockerHandler) VolumeInspect(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.RespondError(c, http.StatusBadRequest, "卷名称不能为空")
		return
	}

	result, err := h.service.InspectVolume(c.Request.Context(), name)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取卷详情失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Volume inspected successfully", result)
}
