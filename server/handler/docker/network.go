package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	dockerPkg "isrvd/pkgs/docker"
	"isrvd/server/helper"
)

// ListNetworks 列出网络
func (h *DockerHandler) ListNetworks(c *gin.Context) {
	result, err := h.service.ListNetworks(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取网络列表失败")
		return
	}

	helper.RespondSuccess(c, "Networks listed successfully", result)
}

// NetworkAction 网络操作
func (h *DockerHandler) NetworkAction(c *gin.Context) {
	var req dockerPkg.NetworkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	if err := h.service.NetworkAction(c.Request.Context(), req.ID, req.Action); err != nil {
		logman.Error("Network action failed", "action", req.Action, "id", req.ID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, req.Action+"网络失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Network "+req.Action+" successfully", nil)
}

// CreateNetwork 创建网络
func (h *DockerHandler) CreateNetwork(c *gin.Context) {
	var req dockerPkg.NetworkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	id, err := h.service.CreateNetwork(c.Request.Context(), req.Name, req.Driver)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "创建网络失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "网络创建成功", gin.H{"id": id, "name": req.Name})
}

// NetworkInspect 获取网络详情
func (h *DockerHandler) NetworkInspect(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "网络ID不能为空")
		return
	}

	result, err := h.service.InspectNetwork(c.Request.Context(), id)
	if err != nil {
		logman.Error("Network inspect failed", "id", id, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取网络详情失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Network inspected successfully", result)
}
