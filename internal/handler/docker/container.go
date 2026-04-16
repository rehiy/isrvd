package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/internal/helper"
	"isrvd/pkgs/docker"
)

// ListContainers 列出容器
func (h *DockerHandler) ListContainers(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"

	result, err := h.service.ListContainers(c.Request.Context(), all)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败")
		return
	}

	helper.RespondSuccess(c, "Containers listed successfully", result)
}

// CreateContainer 创建容器
func (h *DockerHandler) CreateContainer(c *gin.Context) {
	var req docker.ContainerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	id, err := h.service.CreateContainer(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "创建容器失败: "+err.Error())
		return
	}

	shortID := id
	if len(id) > 12 {
		shortID = id[:12]
	}

	helper.RespondSuccess(c, "容器创建成功", gin.H{"id": shortID, "name": req.Name})
}

// UpdateContainerConfig 更新容器配置并重建
func (h *DockerHandler) UpdateContainerConfig(c *gin.Context) {
	var req docker.ContainerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器名称不能为空")
		return
	}

	id, err := h.service.UpdateContainer(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "重建容器失败: "+err.Error())
		return
	}

	shortID := id
	if len(id) > 12 {
		shortID = id[:12]
	}

	helper.RespondSuccess(c, "容器配置更新成功，已重建容器", gin.H{"id": shortID, "name": req.Name})
}

// GetContainerConfig 获取容器配置
func (h *DockerHandler) GetContainerConfig(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器名称不能为空")
		return
	}

	result, err := h.service.GetContainerConfig(c.Request.Context(), name)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "容器配置未找到: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "获取容器配置成功", result)
}

// ContainerStats 获取容器统计信息
func (h *DockerHandler) ContainerStats(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器ID不能为空")
		return
	}

	result, err := h.service.GetContainerStats(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取容器统计信息失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Container stats retrieved", result)
}

// ContainerAction 容器操作
func (h *DockerHandler) ContainerAction(c *gin.Context) {
	var req docker.ContainerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	if err := h.service.ContainerAction(c.Request.Context(), req.ID, req.Action); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, req.Action+"容器失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Container "+req.Action+" successfully", nil)
}

// ContainerLogs 获取容器日志
func (h *DockerHandler) ContainerLogs(c *gin.Context) {
	var req docker.ContainerLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	logs, err := h.service.GetContainerLogs(c.Request.Context(), req.ID, req.Tail)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取日志失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Container logs retrieved", gin.H{"id": req.ID, "logs": logs})
}

// ContainerExec 容器终端 WebSocket 处理
func (h *DockerHandler) ContainerExec(c *gin.Context) {
	conn, err := helper.WsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket upgrade failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "WebSocket 升级失败")
		return
	}
	defer conn.Close()

	containerID := c.Query("id")
	shell := c.DefaultQuery("shell", "/bin/sh")

	if containerID == "" {
		h.sendWsMessage(conn, "[错误: 缺少容器ID]\r\n")
		return
	}

	h.service.ContainerExec(conn, containerID, shell)
}
