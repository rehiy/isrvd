// Package apisix 提供 Apisix 业务服务层
package apisix

import (
	"fmt"

	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
	pkgapisix "isrvd/pkgs/apisix"
)

// Service Apisix 业务服务
type Service struct {
	client *pkgapisix.Client
}

// NewService 创建 Apisix 业务服务
func NewService() (*Service, error) {
	client := registry.ApisixClient
	if client == nil {
		logman.Error("Apisix client not initialized")
		return nil, fmt.Errorf("Apisix 未配置")
	}
	return &Service{client: client}, nil
}

// CheckAvailability 检测 Apisix 可用性
func (s *Service) CheckAvailability() bool {
	if s.client == nil {
		return false
	}
	_, err := s.client.ListRoutes()
	return err == nil
}

// ─── 路由管理 ───

// ListRoutes 获取所有路由列表
func (s *Service) ListRoutes() (any, error) {
	return s.client.ListRoutes()
}

// GetRoute 获取单条路由详情
func (s *Service) GetRoute(routeID string) (any, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	return s.client.GetRoute(routeID)
}

// CreateRoute 创建路由
func (s *Service) CreateRoute(req pkgapisix.Route) (any, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	if req.URI == "" && len(req.URIs) == 0 {
		return nil, fmt.Errorf("URI 或 URIs 不能为空")
	}
	return s.client.CreateRoute(req)
}

// UpdateRoute 更新路由
func (s *Service) UpdateRoute(routeID string, req pkgapisix.Route) (any, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	return s.client.UpdateRoute(routeID, req)
}

// PatchRouteStatus 更新路由启用/禁用状态
func (s *Service) PatchRouteStatus(routeID string, status int) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	if status != 0 && status != 1 {
		return fmt.Errorf("状态值必须为 1（启用）或 0（禁用）")
	}
	return s.client.PatchRouteStatus(routeID, status)
}

// DeleteRoute 删除路由
func (s *Service) DeleteRoute(routeID string) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	return s.client.DeleteRoute(routeID)
}

// ─── Consumer 管理 ───

// ListConsumers 获取 Consumer 列表
func (s *Service) ListConsumers() (any, error) {
	return s.client.ListConsumers()
}

// CreateConsumer 创建 Consumer
func (s *Service) CreateConsumer(username, desc string) (any, error) {
	return s.client.CreateConsumer(username, desc)
}

// UpdateConsumerDesc 更新 Consumer 描述
func (s *Service) UpdateConsumerDesc(username, desc string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return s.client.UpdateConsumerDesc(username, desc)
}

// DeleteConsumer 删除 Consumer
func (s *Service) DeleteConsumer(username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return s.client.DeleteConsumer(username)
}

// ─── 白名单管理 ───

// GetWhitelist 获取白名单
func (s *Service) GetWhitelist() (any, error) {
	return s.client.GetRouteWhitelist()
}

// RevokeWhitelist 移除白名单
func (s *Service) RevokeWhitelist(routeID, consumerName string) error {
	return s.client.RemoveConsumerFromRouteWhitelist(routeID, consumerName)
}

// ─── 辅助资源 ───

// ListPluginConfigs 获取 Plugin Config 列表
func (s *Service) ListPluginConfigs() (any, error) {
	return s.client.ListPluginConfigs()
}

// ListUpstreams 获取 Upstream 列表
func (s *Service) ListUpstreams() (any, error) {
	return s.client.ListUpstreams()
}

// ListPlugins 获取可用插件列表
func (s *Service) ListPlugins() (any, error) {
	return s.client.ListPlugins()
}
