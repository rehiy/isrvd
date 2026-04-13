package apisix

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

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
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", routes)
}

// GetRoute 获取单条路由详情
func (h *Handler) GetRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	route, err := h.client.GetRoute(c.Request.Context(), routeID)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", route)
}

// CreateRoute 创建路由
func (h *Handler) CreateRoute(c *gin.Context) {
	var req RouteUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	if req.URI == "" && len(req.URIs) == 0 {
		helper.RespondError(c, http.StatusBadRequest, "URI 或 URIs 至少填写一个")
		return
	}
	route, err := h.client.CreateRoute(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "创建成功", route)
}

// UpdateRoute 更新路由
func (h *Handler) UpdateRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req RouteUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	route, err := h.client.UpdateRoute(c.Request.Context(), routeID, req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "更新成功", route)
}

// PatchRouteStatus 更新路由启用/禁用状态
func (h *Handler) PatchRouteStatus(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if req.Status != 0 && req.Status != 1 {
		helper.RespondError(c, http.StatusBadRequest, "status 只能为 1（启用）或 0（禁用）")
		return
	}
	if err := h.client.PatchRouteStatus(c.Request.Context(), routeID, req.Status); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "状态更新成功", nil)
}

// ========================
// Consumer 管理接口
// ========================

// ListConsumers 获取 Consumer 列表
func (h *Handler) ListConsumers(c *gin.Context) {
	consumers, err := h.client.ListConsumers(c.Request.Context())
	if err != nil {
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
		helper.RespondError(c, http.StatusBadRequest, "用户名为必填项")
		return
	}
	apiKey := generateAPIKey()
	if err := h.client.CreateOrUpdateConsumer(c.Request.Context(), req.Username, apiKey, req.Description); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "创建成功", ConsumerDTO{
		Username:    req.Username,
		APIKey:      apiKey,
		Description: req.Description,
	})
}

// UpdateConsumer 更新 Consumer 描述
func (h *Handler) UpdateConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	// 保持原有的 API Key 不变
	apiKey, err := h.client.GetConsumerRawKey(c.Request.Context(), username)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}
	if err := h.client.CreateOrUpdateConsumer(c.Request.Context(), username, apiKey, req.Description); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "更新成功", ConsumerDTO{
		Username:    username,
		Description: req.Description,
	})
}

// DeleteConsumer 删除 Consumer
func (h *Handler) DeleteConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if err := h.client.DeleteConsumer(c.Request.Context(), username); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "删除成功", nil)
}

// DeleteRoute 删除路由
func (h *Handler) DeleteRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	if err := h.client.DeleteRoute(c.Request.Context(), routeID); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "删除成功", nil)
}

// ========================
// 白名单管理接口
// ========================

// GetWhitelist 获取 Apisix 中实际生效的 consumer-restriction 白名单
func (h *Handler) GetWhitelist(c *gin.Context) {
	result, err := h.client.GetRouteWhitelist(c.Request.Context())
	if err != nil {
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
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 和用户名为必填项")
		return
	}
	if err := h.client.RemoveConsumerFromRouteWhitelist(c.Request.Context(), req.RouteID, req.ConsumerName); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "撤销成功", nil)
}

// ========================
// 辅助资源接口
// ========================

// ListPluginConfigs 获取 Plugin Config 列表
func (h *Handler) ListPluginConfigs(c *gin.Context) {
	list, err := h.client.ListPluginConfigs(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListUpstreams 获取 Upstream 列表
func (h *Handler) ListUpstreams(c *gin.Context) {
	list, err := h.client.ListUpstreams(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListPlugins 获取可用插件列表
func (h *Handler) ListPlugins(c *gin.Context) {
	list, err := h.client.ListPlugins(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}
