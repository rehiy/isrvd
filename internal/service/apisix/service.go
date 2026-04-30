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

// CreateConsumer 创建 Consumer，支持传入完整 plugins
func (s *Service) CreateConsumer(username, desc string, plugins map[string]any) (any, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	return s.client.CreateConsumer(username, desc, plugins)
}

// UpdateConsumer 更新 Consumer（支持 plugins，自动替换脱敏值）
func (s *Service) UpdateConsumer(username, desc string, plugins map[string]any) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return s.client.UpdateConsumer(username, desc, plugins)
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

// ─── Upstream 管理 ───

// ListUpstreams 获取 Upstream 列表
func (s *Service) ListUpstreams() (any, error) {
	return s.client.ListUpstreams()
}

// GetUpstream 获取单条 Upstream 详情
func (s *Service) GetUpstream(upstreamID string) (any, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	return s.client.GetUpstream(upstreamID)
}

// CreateUpstream 创建 Upstream
func (s *Service) CreateUpstream(req pkgapisix.Upstream) (any, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !pkgapisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	return s.client.CreateUpstream(req)
}

// UpdateUpstream 更新 Upstream
func (s *Service) UpdateUpstream(upstreamID string, req pkgapisix.Upstream) (any, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !pkgapisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	return s.client.UpdateUpstream(upstreamID, req)
}

// DeleteUpstream 删除 Upstream
func (s *Service) DeleteUpstream(upstreamID string) error {
	if upstreamID == "" {
		return fmt.Errorf("Upstream ID 不能为空")
	}
	return s.client.DeleteUpstream(upstreamID)
}

// ─── SSL 证书管理 ───

// ListSSLs 获取 SSL 证书列表
func (s *Service) ListSSLs() (any, error) {
	return s.client.ListSSLs()
}

// GetSSL 获取单个 SSL 证书详情
func (s *Service) GetSSL(sslID string) (any, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	return s.client.GetSSL(sslID)
}

// CreateSSL 创建 SSL 证书
func (s *Service) CreateSSL(req pkgapisix.SSL) (any, error) {
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	if req.Cert == "" {
		return nil, fmt.Errorf("证书内容不能为空")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("私钥内容不能为空")
	}
	return s.client.CreateSSL(req)
}

// UpdateSSL 更新 SSL 证书
func (s *Service) UpdateSSL(sslID string, req pkgapisix.SSL) (any, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	return s.client.UpdateSSL(sslID, req)
}

// DeleteSSL 删除 SSL 证书
func (s *Service) DeleteSSL(sslID string) error {
	if sslID == "" {
		return fmt.Errorf("SSL 证书 ID 不能为空")
	}
	return s.client.DeleteSSL(sslID)
}

// ─── 辅助资源 ───

// ListPluginConfigs 获取 Plugin Config 列表
func (s *Service) ListPluginConfigs() (any, error) {
	return s.client.ListPluginConfigs()
}

// GetPluginConfig 获取单个 Plugin Config 详情
func (s *Service) GetPluginConfig(configID string) (any, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	return s.client.GetPluginConfig(configID)
}

// CreatePluginConfig 创建 Plugin Config
func (s *Service) CreatePluginConfig(req pkgapisix.PluginConfig) (any, error) {
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	return s.client.CreatePluginConfig(req)
}

// UpdatePluginConfig 更新 Plugin Config
func (s *Service) UpdatePluginConfig(configID string, req pkgapisix.PluginConfig) (any, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	return s.client.UpdatePluginConfig(configID, req)
}

// DeletePluginConfig 删除 Plugin Config
func (s *Service) DeletePluginConfig(configID string) error {
	if configID == "" {
		return fmt.Errorf("Plugin Config ID 不能为空")
	}
	return s.client.DeletePluginConfig(configID)
}

// ListPlugins 获取可用插件列表
func (s *Service) ListPlugins() (any, error) {
	return s.client.ListPlugins()
}
