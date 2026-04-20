package swarm

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	"isrvd/internal/registry"
	"isrvd/pkgs/swarm"
)

// SwarmHandler Swarm 处理器
type SwarmHandler struct {
	manager *swarm.SwarmManager
}

// NewSwarmHandler 创建 Swarm 处理器
func NewSwarmHandler() *SwarmHandler {
	return &SwarmHandler{
		manager: registry.SwarmManager,
	}
}

// CheckAvailability 检测 Swarm 可用性，实现 system.SwarmAvailabilityChecker 接口
func (h *SwarmHandler) CheckAvailability(ctx context.Context) bool {
	if h.manager == nil {
		return false
	}
	_, err := h.manager.GetClient().SwarmInspect(ctx)
	return err == nil
}

// SwarmInfo 获取 Swarm 集群概览
func (h *SwarmHandler) SwarmInfo(c *gin.Context) {
	info, err := h.manager.GetSwarmInfo(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取 Swarm 信息失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Swarm info retrieved", info)
}

// SwarmListNodes 获取节点列表
func (h *SwarmHandler) SwarmListNodes(c *gin.Context) {
	result, err := h.manager.ListNodes(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取节点列表失败")
		return
	}

	helper.RespondSuccess(c, "Nodes listed", result)
}

// SwarmNodeAction 节点操作
func (h *SwarmHandler) SwarmNodeAction(c *gin.Context) {
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.manager.NodeAction(c.Request.Context(), req.ID, req.Action); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "节点操作失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Node updated", nil)
}

// SwarmListServices 获取服务列表
func (h *SwarmHandler) SwarmListServices(c *gin.Context) {
	result, err := h.manager.ListServices(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取服务列表失败")
		return
	}

	helper.RespondSuccess(c, "Services listed", result)
}

// SwarmServiceAction 服务操作
func (h *SwarmHandler) SwarmServiceAction(c *gin.Context) {
	var req struct {
		ID       string  `json:"id"`
		Action   string  `json:"action"`
		Replicas *uint64 `json:"replicas,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.manager.ServiceAction(c.Request.Context(), req.ID, req.Action, req.Replicas); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "服务操作失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service "+req.Action+" successfully", nil)
}

// SwarmListTasks 获取任务列表
func (h *SwarmHandler) SwarmListTasks(c *gin.Context) {
	serviceID := c.Query("serviceID")

	result, err := h.manager.ListTasks(c.Request.Context(), serviceID)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取任务列表失败")
		return
	}

	helper.RespondSuccess(c, "Tasks listed", result)
}

// SwarmCreateService 创建服务
func (h *SwarmHandler) SwarmCreateService(c *gin.Context) {
	var req swarm.SwarmCreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	id, err := h.manager.CreateService(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "服务创建失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service created", gin.H{"id": id})
}

// SwarmForceUpdateService 强制重新部署服务
func (h *SwarmHandler) SwarmForceUpdateService(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.manager.ForceUpdateService(c.Request.Context(), req.ID); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "强制重部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service force updated", nil)
}

// SwarmServiceLogs 获取服务日志
func (h *SwarmHandler) SwarmServiceLogs(c *gin.Context) {
	serviceID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")
	if serviceID == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少服务 ID")
		return
	}

	logs, err := h.manager.GetServiceLogs(c.Request.Context(), serviceID, tail)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取日志失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Logs retrieved", gin.H{"logs": logs})
}

// SwarmInspectNode 获取节点详情
func (h *SwarmHandler) SwarmInspectNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少节点 ID")
		return
	}

	result, err := h.manager.InspectNode(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取节点详情失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Node inspected", result)
}

// SwarmInspectService 获取服务详情
func (h *SwarmHandler) SwarmInspectService(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少服务 ID")
		return
	}

	result, err := h.manager.InspectService(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取服务详情失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service inspected", result)
}
