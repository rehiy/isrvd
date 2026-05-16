// Package apisix 提供 Apisix 业务服务层
package apisix

import (
	"fmt"

	"github.com/rehiy/libgo/logman"

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
	// 验证连通性，服务不可达时拒绝初始化
	if _, err := client.RouteList(); err != nil {
		return nil, fmt.Errorf("Apisix 不可达: %w", err)
	}
	return &Service{client: client}, nil
}

// CheckAvailability 检测 Apisix 可用性
func (s *Service) CheckAvailability() bool {
	if s.client == nil {
		return false
	}
	_, err := s.client.RouteList()
	return err == nil
}

// ─── 路由管理 ───

// RouteList 获取所有路由列表
func (s *Service) RouteList() ([]pkgapisix.Route, error) {
	return s.client.RouteList()
}

// RouteInspect 获取单条路由详情
func (s *Service) RouteInspect(routeID string) (*pkgapisix.Route, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	return s.client.RouteInspect(routeID)
}

// RouteCreate 创建路由
func (s *Service) RouteCreate(req pkgapisix.Route) (*pkgapisix.Route, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	if req.URI == "" && len(req.URIs) == 0 {
		return nil, fmt.Errorf("URI 或 URIs 不能为空")
	}
	return s.client.RouteCreate(req)
}

// RouteUpdate 更新路由
func (s *Service) RouteUpdate(routeID string, req pkgapisix.Route) (*pkgapisix.Route, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	return s.client.RouteUpdate(routeID, req)
}

// RouteStatusPatch 更新路由启用/禁用状态
func (s *Service) RouteStatusPatch(routeID string, status int) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	if status != 0 && status != 1 {
		return fmt.Errorf("状态值必须为 1（启用）或 0（禁用）")
	}
	return s.client.RouteStatusPatch(routeID, status)
}

// RouteDelete 删除路由
func (s *Service) RouteDelete(routeID string) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	return s.client.RouteDelete(routeID)
}

// ─── Consumer 管理 ───

// ConsumerList 获取 Consumer 列表
func (s *Service) ConsumerList() ([]pkgapisix.Consumer, error) {
	return s.client.ConsumerList()
}

// ConsumerCreate 创建 Consumer，支持传入完整 plugins
func (s *Service) ConsumerCreate(username, desc string, plugins map[string]any) (*pkgapisix.Consumer, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	return s.client.ConsumerCreate(username, desc, plugins)
}

// ConsumerUpdate 更新 Consumer（支持 plugins，自动替换脱敏值）
func (s *Service) ConsumerUpdate(username, desc string, plugins map[string]any) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return s.client.ConsumerUpdate(username, desc, plugins)
}

// ConsumerDelete 删除 Consumer
func (s *Service) ConsumerDelete(username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return s.client.ConsumerDelete(username)
}

// ─── 白名单管理 ───

// WhitelistList 获取白名单
func (s *Service) WhitelistList() ([]pkgapisix.Route, error) {
	return s.client.RouteWhitelist()
}

// WhitelistRevoke 移除白名单
func (s *Service) WhitelistRevoke(routeID, consumerName string) error {
	return s.client.RouteWhitelistRevoke(routeID, consumerName)
}

// ─── Upstream 管理 ───

// UpstreamList 获取 Upstream 列表
func (s *Service) UpstreamList() ([]pkgapisix.Upstream, error) {
	return s.client.UpstreamList()
}

// UpstreamInspect 获取单条 Upstream 详情
func (s *Service) UpstreamInspect(upstreamID string) (*pkgapisix.Upstream, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	return s.client.UpstreamInspect(upstreamID)
}

// UpstreamCreate 创建 Upstream
func (s *Service) UpstreamCreate(req pkgapisix.Upstream) (*pkgapisix.Upstream, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !pkgapisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	return s.client.UpstreamCreate(req)
}

// UpstreamUpdate 更新 Upstream
func (s *Service) UpstreamUpdate(upstreamID string, req pkgapisix.Upstream) (*pkgapisix.Upstream, error) {
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
	return s.client.UpstreamUpdate(upstreamID, req)
}

// UpstreamDelete 删除 Upstream
func (s *Service) UpstreamDelete(upstreamID string) error {
	if upstreamID == "" {
		return fmt.Errorf("Upstream ID 不能为空")
	}
	return s.client.UpstreamDelete(upstreamID)
}

// ─── SSL 证书管理 ───

// SSLList 获取 SSL 证书列表
func (s *Service) SSLList() ([]pkgapisix.SSL, error) {
	return s.client.SSLList()
}

// SSLInspect 获取单个 SSL 证书详情
func (s *Service) SSLInspect(sslID string) (*pkgapisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	return s.client.SSLInspect(sslID)
}

// SSLCreate 创建 SSL 证书
func (s *Service) SSLCreate(req pkgapisix.SSL) (*pkgapisix.SSL, error) {
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	if req.Cert == "" {
		return nil, fmt.Errorf("证书内容不能为空")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("私鑰内容不能为空")
	}
	return s.client.SSLCreate(req)
}

// SSLUpdate 更新 SSL 证书
func (s *Service) SSLUpdate(sslID string, req pkgapisix.SSL) (*pkgapisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	return s.client.SSLUpdate(sslID, req)
}

// SSLDelete 删除 SSL 证书
func (s *Service) SSLDelete(sslID string) error {
	if sslID == "" {
		return fmt.Errorf("SSL 证书 ID 不能为空")
	}
	return s.client.SSLDelete(sslID)
}

// ─── 辅助资源 ───

// PluginConfigList 获取 Plugin Config 列表
func (s *Service) PluginConfigList() ([]pkgapisix.PluginConfig, error) {
	return s.client.PluginConfigList()
}

// PluginConfigInspect 获取单个 Plugin Config 详情
func (s *Service) PluginConfigInspect(configID string) (*pkgapisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	return s.client.PluginConfigInspect(configID)
}

// PluginConfigCreate 创建 Plugin Config
func (s *Service) PluginConfigCreate(req pkgapisix.PluginConfig) (*pkgapisix.PluginConfig, error) {
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	return s.client.PluginConfigCreate(req)
}

// PluginConfigUpdate 更新 Plugin Config
func (s *Service) PluginConfigUpdate(configID string, req pkgapisix.PluginConfig) (*pkgapisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	return s.client.PluginConfigUpdate(configID, req)
}

// PluginConfigDelete 删除 Plugin Config
func (s *Service) PluginConfigDelete(configID string) error {
	if configID == "" {
		return fmt.Errorf("Plugin Config ID 不能为空")
	}
	return s.client.PluginConfigDelete(configID)
}

// PluginList 获取可用插件列表
func (s *Service) PluginList() (any, error) {
	return s.client.PluginList()
}
