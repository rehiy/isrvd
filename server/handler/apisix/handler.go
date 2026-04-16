package apisix

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
	"isrvd/pkgs/apisix"
	"isrvd/server/helper"
)

// Handler Apisix 管理处理器
type Handler struct {
	client *apisix.Client
}

// NewHandler 创建 Apisix 处理器
func NewHandler() (*Handler, error) {
	client := registry.ApisixClient
	if client == nil {
		logman.Error("Apisix client not initialized")
		return nil, fmt.Errorf("Apisix 未配置")
	}

	return &Handler{
		client: client,
	}, nil
}

// CheckAvailability 检测 APISIX 可用性，实现 system.ApisixAvailabilityChecker 接口
func (h *Handler) CheckAvailability() bool {
	if h.client == nil {
		return false
	}
	_, err := h.client.ListRoutes()
	return err == nil
}

// ========================
// 路由管理接口
// ========================

// ListRoutes 获取所有路由列表
func (h *Handler) ListRoutes(c *gin.Context) {
	routes, err := h.client.ListRoutes()
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
	route, err := h.client.GetRoute(routeID)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", route)
}

// CreateRoute 创建路由
func (h *Handler) CreateRoute(c *gin.Context) {
	var req apisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	if req.URI == "" && len(req.URIs) == 0 {
		helper.RespondError(c, http.StatusBadRequest, "URI 或 URIs 不能为空")
		return
	}
	route, err := h.client.CreateRoute(req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route created successfully", route)
}

// UpdateRoute 更新路由
func (h *Handler) UpdateRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req apisix.Route
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由名称不能为空")
		return
	}
	route, err := h.client.UpdateRoute(routeID, req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Route updated successfully", route)
}

// PatchRouteStatus 更新路由启用/禁用状态
func (h *Handler) PatchRouteStatus(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	var req PatchRouteStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if req.Status != 0 && req.Status != 1 {
		helper.RespondError(c, http.StatusBadRequest, "状态值必须为 1（启用）或 0（禁用）")
		return
	}
	if err := h.client.PatchRouteStatus(routeID, req.Status); err != nil {
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
	consumers, err := h.client.ListConsumers()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", consumers)
}

// CreateConsumer 创建 Consumer
func (h *Handler) CreateConsumer(c *gin.Context) {
	var req CreateConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	consumer, err := h.client.CreateConsumer(req.Username, req.Desc)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer created successfully", consumer)
}

// UpdateConsumer 更新 Consumer 描述
func (h *Handler) UpdateConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	var req UpdateConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}
	if err := h.client.UpdateConsumerDesc(username, req.Desc); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer updated successfully", gin.H{
		"username": username,
		"desc":     req.Desc,
	})
}

// DeleteConsumer 删除 Consumer
func (h *Handler) DeleteConsumer(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if err := h.client.DeleteConsumer(username); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Consumer deleted successfully", nil)
}

// DeleteRoute 删除路由
func (h *Handler) DeleteRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 不能为空")
		return
	}
	if err := h.client.DeleteRoute(routeID); err != nil {
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
	result, err := h.client.GetRouteWhitelist()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", result)
}

// RevokeWhitelist 从 Apisix 路由白名单中移除 consumer
func (h *Handler) RevokeWhitelist(c *gin.Context) {
	var req RevokeWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "路由 ID 和消费者名称不能为空")
		return
	}
	if err := h.client.RemoveConsumerFromRouteWhitelist(req.RouteID, req.ConsumerName); err != nil {
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
	list, err := h.client.ListPluginConfigs()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListUpstreams 获取 Upstream 列表
func (h *Handler) ListUpstreams(c *gin.Context) {
	list, err := h.client.ListUpstreams()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}

// ListPlugins 获取可用插件列表
func (h *Handler) ListPlugins(c *gin.Context) {
	list, err := h.client.ListPlugins()
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "", list)
}
