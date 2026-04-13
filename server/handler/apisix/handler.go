package apisix

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"
)

// Handler Apisix 管理处理器
type Handler struct {
	client *Client
}

// NewHandler 创建 Apisix 处理器
func NewHandler() (*Handler, error) {
	if config.Apisix.AdminURL == "" {
		return nil, fmt.Errorf("Apisix adminUrl 未配置")
	}
	return &Handler{
		client: NewClient(config.Apisix),
	}, nil
}

// ========================
// 路由管理接口
// ========================

// ListRoutes 获取所有路由列表
func (h *Handler) ListRoutes(c *gin.Context) {
	routes, err := h.client.ListRoutes(c.Request.Context())
	if err != nil {
		logman.Error("List routes failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", routes)
}

// GetRoute 获取单条路由详情
func (h *Handler) GetRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		logman.Error("Get route failed", "error", "route ID is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	route, err := h.client.GetRoute(c.Request.Context(), routeID)
	if err != nil {
		logman.Error("Get route failed", "routeID", routeID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", route)
}

// CreateRoute 创建路由
func (h *Handler) CreateRoute(c *gin.Context) {
	var req RouteUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create route failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Name == "" {
		logman.Error("Create route failed", "error", "route name is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	if req.URI == "" && len(req.URIs) == 0 {
		logman.Error("Create route failed", "error", "URI or URIs is required")
		helper.RespondError(c, http.StatusBadRequest, "URI 或 URIs 不能为空")
		return
	}
	route, err := h.client.CreateRoute(c.Request.Context(), req)
	if err != nil {
		logman.Error("Create route failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route created successfully", route)
}

// UpdateRoute 更新路由
func (h *Handler) UpdateRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		logman.Error("Update route failed", "error", "route ID is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req RouteUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Update route failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Name == "" {
		logman.Error("Update route failed", "error", "route name is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	route, err := h.client.UpdateRoute(c.Request.Context(), routeID, req)
	if err != nil {
		logman.Error("Update route failed", "routeID", routeID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route updated successfully", route)
}

// PatchRouteStatus 更新路由启用/禁用状态
func (h *Handler) PatchRouteStatus(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		logman.Error("Patch route status failed", "error", "route ID is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Patch route status failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Status != 0 && req.Status != 1 {
		logman.Error("Patch route status failed", "error", "invalid status value", "status", req.Status)
		helper.RespondError(c, http.StatusBadRequest, "状态值必须为 1（启用）或 0（禁用）")
		return
	}
	if err := h.client.PatchRouteStatus(c.Request.Context(), routeID, req.Status); err != nil {
		logman.Error("Patch route status failed", "routeID", routeID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route status updated successfully", nil)
}

// ========================
// Consumer 管理接口
// ========================

// ListConsumers 获取 Consumer 列表
func (h *Handler) ListConsumers(c *gin.Context) {
	consumers, err := h.client.ListConsumers(c.Request.Context())
	if err != nil {
		logman.Error("List consumers failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", consumers)
}

// CreateConsumer 创建 Consumer
func (h *Handler) CreateConsumer(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create consumer failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	apiKey := generateAPIKey()
	if err := h.client.CreateOrUpdateConsumer(c.Request.Context(), req.Username, apiKey, req.Description); err != nil {
		logman.Error("Create consumer failed", "username", req.Username, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer created successfully",		Username:    req.Username,
		APIKey:      apiKey,
		Description: req.Description,
	})
}

// UpdateConsumer 更新 Consumer 描述
func (h *Handler) UpdateConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		logman.Error("Update consumer failed", "error", "username is empty")
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Update consumer failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	// 保持原有的 API Key 不变
	apiKey, err := h.client.GetConsumerRawKey(c.Request.Context(), username)
	if err != nil {
		logman.Error("Get consumer key failed", "username", username, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取消费者信息失败: "+err.Error())
		return
	}
	if err := h.client.CreateOrUpdateConsumer(c.Request.Context(), username, apiKey, req.Description); err != nil {
		logman.Error("Update consumer failed", "username", username, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer updated successfully",		Username:    username,
		Description: req.Description,
	})
}

// DeleteConsumer 删除 Consumer
func (h *Handler) DeleteConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		logman.Error("Delete consumer failed", "error", "username is empty")
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if err := h.client.DeleteConsumer(c.Request.Context(), username); err != nil {
		logman.Error("Delete consumer failed", "username", username, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer deleted successfully", nil)
}

// DeleteRoute 删除路由
func (h *Handler) DeleteRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		logman.Error("Delete route failed", "error", "route ID is empty")
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	if err := h.client.DeleteRoute(c.Request.Context(), routeID); err != nil {
		logman.Error("Delete route failed", "routeID", routeID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route deleted successfully", nil)
}

// ========================
// 白名单管理接口
// ========================

// GetWhitelist 获取 Apisix 中实际生效的 consumer-restriction 白名单
func (h *Handler) GetWhitelist(c *gin.Context) {
	result, err := h.client.GetRouteWhitelist(c.Request.Context())
	if err != nil {
		logman.Error("Get whitelist failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

// RevokeWhitelist 从 Apisix 路由白名单中移除 consumer
func (h *Handler) RevokeWhitelist(c *gin.Context) {
	var req struct {
		RouteID      string `json:"route_id" binding:"required"`
		ConsumerName string `json:"consumer_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Revoke whitelist failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 和消费者名称不能为空")
		return
	}
	if err := h.client.RemoveConsumerFromRouteWhitelist(c.Request.Context(), req.RouteID, req.ConsumerName); err != nil {
		logman.Error("Revoke whitelist failed", "routeID", req.RouteID, "consumer", req.ConsumerName, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Whitelist revoked successfully", nil)
}

// ========================
// 辅助资源接口
// ========================

// ListPluginConfigs 获取 Plugin Config 列表
func (h *Handler) ListPluginConfigs(c *gin.Context) {
	list, err := h.client.ListPluginConfigs(c.Request.Context())
	if err != nil {
		logman.Error("List plugin configs failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListUpstreams 获取 Upstream 列表
func (h *Handler) ListUpstreams(c *gin.Context) {
	list, err := h.client.ListUpstreams(c.Request.Context())
	if err != nil {
		logman.Error("List upstreams failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListPlugins 获取可用插件列表
func (h *Handler) ListPlugins(c *gin.Context) {
	list, err := h.client.ListPlugins(c.Request.Context())
	if err != nil {
		logman.Error("List plugins failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}
