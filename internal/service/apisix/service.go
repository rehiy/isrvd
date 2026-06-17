// Package apisix 提供 Apisix 业务服务层
package apisix

import (
	"context"
	"fmt"
	"strings"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/strutil"

	"isrvd/internal/registry"
	pkgApisix "isrvd/pkgs/apisix"
)

// Service Apisix 业务服务
type Service struct {
	client *pkgApisix.Client
}

// NewService 创建 Apisix 业务服务
func NewService() (*Service, error) {
	client := registry.ApisixClient
	if client == nil {
		logman.Warn("Apisix client not initialized")
		return nil, fmt.Errorf("Apisix 未配置")
	}
	// 验证连通性，服务不可达时拒绝初始化
	if _, err := client.RouteList(context.Background()); err != nil {
		return nil, fmt.Errorf("Apisix 不可达: %w", err)
	}
	return &Service{client: client}, nil
}

// CheckAvailability 检测 Apisix 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.client == nil {
		return false
	}
	_, err := s.client.RouteList(ctx)
	return err == nil
}

// ─── 路由管理 ───

// RouteList 获取所有路由列表
func (s *Service) RouteList(ctx context.Context) ([]pkgApisix.Route, error) {
	list, err := s.client.RouteList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取路由列表失败: %w", err)
	}
	return list, nil
}

// RouteInspect 获取单条路由详情
func (s *Service) RouteInspect(ctx context.Context, routeID string) (*pkgApisix.Route, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	route, err := s.client.RouteInspect(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("获取路由详情失败: %w", err)
	}
	return route, nil
}

// RouteCreate 创建路由
func (s *Service) RouteCreate(ctx context.Context, req pkgApisix.Route) (*pkgApisix.Route, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	if req.URI == "" && len(req.URIs) == 0 {
		return nil, fmt.Errorf("URI 或 URIs 不能为空")
	}
	route, err := s.client.RouteCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建路由失败: %w", err)
	}
	return route, nil
}

// RouteUpdate 更新路由
func (s *Service) RouteUpdate(ctx context.Context, routeID string, req pkgApisix.Route) (*pkgApisix.Route, error) {
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("路由名称不能为空")
	}
	route, err := s.client.RouteUpdate(ctx, routeID, req)
	if err != nil {
		return nil, fmt.Errorf("更新路由失败: %w", err)
	}
	return route, nil
}

// RouteStatusPatch 更新路由启用/禁用状态
func (s *Service) RouteStatusPatch(ctx context.Context, routeID string, status int) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	if status != 0 && status != 1 {
		return fmt.Errorf("状态值必须为 1（启用）或 0（禁用）")
	}
	if err := s.client.RouteStatusPatch(ctx, routeID, status); err != nil {
		return fmt.Errorf("更新路由状态失败: %w", err)
	}
	return nil
}

// RouteDelete 删除路由
func (s *Service) RouteDelete(ctx context.Context, routeID string) error {
	if routeID == "" {
		return fmt.Errorf("路由 ID 不能为空")
	}
	if err := s.client.RouteDelete(ctx, routeID); err != nil {
		return fmt.Errorf("删除路由失败: %w", err)
	}
	return nil
}

// ─── Consumer 管理 ───

// ConsumerList 获取 Consumer 列表
func (s *Service) ConsumerList(ctx context.Context) ([]pkgApisix.Consumer, error) {
	list, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	return list, nil
}

// ConsumerCreate 创建 Consumer，支持传入完整 plugins
func (s *Service) ConsumerCreate(ctx context.Context, username, desc string, plugins map[string]any) (*pkgApisix.Consumer, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	consumer, err := s.client.ConsumerCreate(ctx, username, desc, plugins)
	if err != nil {
		return nil, fmt.Errorf("创建消费者失败: %w", err)
	}
	return consumer, nil
}

// ConsumerUpdate 更新 Consumer（支持 plugins，自动替换脱敏值）
func (s *Service) ConsumerUpdate(ctx context.Context, username, desc string, plugins map[string]any) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if err := s.client.ConsumerUpdate(ctx, username, desc, plugins); err != nil {
		return fmt.Errorf("更新消费者失败: %w", err)
	}
	return nil
}

// ConsumerDelete 删除 Consumer
func (s *Service) ConsumerDelete(ctx context.Context, username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if err := s.client.ConsumerDelete(ctx, username); err != nil {
		return fmt.Errorf("删除消费者失败: %w", err)
	}
	return nil
}

// ─── 白名单管理 ───

// WhitelistCreateRequest 配置白名单路由请求
type WhitelistCreateRequest struct {
	RouteID   string         `json:"route_id"`  // 路由 ID
	Consumers []string       `json:"consumers"` // 白名单 Consumer 列表
	KeyAuth   map[string]any `json:"key_auth"`  // Key Auth 插件配置
}

// WhitelistCreate 为已有路由配置 Consumer 白名单
func (s *Service) WhitelistCreate(ctx context.Context, req WhitelistCreateRequest) (*pkgApisix.Route, error) {
	routeID := strings.TrimSpace(req.RouteID)
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if len(req.Consumers) == 0 {
		return nil, fmt.Errorf("白名单用户不能为空")
	}
	if len(req.KeyAuth) == 0 {
		return nil, fmt.Errorf("key-auth 配置不能为空")
	}
	header, _ := req.KeyAuth["header"].(string)
	if strings.TrimSpace(header) == "" {
		return nil, fmt.Errorf("key-auth 请求头名称不能为空")
	}

	// 去重并过滤空用户名
	seen := make(map[string]struct{}, len(req.Consumers))
	consumers := make([]string, 0, len(req.Consumers))
	for _, name := range req.Consumers {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		consumers = append(consumers, name)
	}
	if len(consumers) == 0 {
		return nil, fmt.Errorf("白名单用户不能为空")
	}

	existingConsumers, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	existingConsumerMap := make(map[string]struct{}, len(existingConsumers))
	for _, consumer := range existingConsumers {
		existingConsumerMap[consumer.Username] = struct{}{}
	}
	for _, consumer := range consumers {
		if _, ok := existingConsumerMap[consumer]; !ok {
			return nil, fmt.Errorf("Consumer %s 不存在，请先创建该 Consumer", consumer)
		}
	}
	if err := s.client.RouteConsumerRestrictionUpdate(ctx, routeID, consumers, req.KeyAuth); err != nil {
		return nil, fmt.Errorf("配置白名单失败: %w", err)
	}
	return s.RouteInspect(ctx, routeID)
}

// WhitelistUserCreateRequest 新建用户并加入白名单请求
type WhitelistUserCreateRequest struct {
	RouteID  string         `json:"route_id"` // 目标路由 ID
	Username string         `json:"username"` // Consumer 用户名
	Key      string         `json:"key"`      // key-auth 密钥
	KeyAuth  map[string]any `json:"key_auth"` // key-auth 插件的附加配置
}

// WhitelistUserCreate 原子操作：创建 Consumer（含 key-auth）并加入路由白名单。
// 若 Consumer 已存在则直接复用，不报错，保证接口幂等。
func (s *Service) WhitelistUserCreate(ctx context.Context, req WhitelistUserCreateRequest) (*pkgApisix.Route, error) {
	routeID := strings.TrimSpace(req.RouteID)
	username := strings.TrimSpace(req.Username)
	key := strings.TrimSpace(req.Key)

	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	if key == "" {
		return nil, fmt.Errorf("key-auth key 不能为空")
	}
	if len(req.KeyAuth) == 0 {
		return nil, fmt.Errorf("key-auth 配置不能为空")
	}
	header, _ := req.KeyAuth["header"].(string)
	if strings.TrimSpace(header) == "" {
		return nil, fmt.Errorf("key-auth 请求头名称不能为空")
	}

	// 步骤一：创建 Consumer，若已存在则跳过
	existingConsumers, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	exists := false
	for _, c := range existingConsumers {
		if c.Username == username {
			exists = true
			break
		}
	}
	if !exists {
		plugins := map[string]any{
			"key-auth": map[string]any{"key": key},
		}
		if _, err := s.client.ConsumerCreate(ctx, username, "", plugins); err != nil {
			return nil, fmt.Errorf("创建消费者失败: %w", err)
		}
	}

	// 步骤二：将 Consumer 加入路由白名单
	route, err := s.client.RouteInspect(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("获取路由详情失败: %w", err)
	}
	consumers := route.Consumers
	for _, c := range consumers {
		if c == username {
			// 已在白名单中，直接返回当前路由
			return route, nil
		}
	}
	consumers = append(consumers, username)
	if err := s.client.RouteConsumerRestrictionUpdate(ctx, routeID, consumers, req.KeyAuth); err != nil {
		return nil, fmt.Errorf("配置白名单失败: %w", err)
	}
	return s.RouteInspect(ctx, routeID)
}

// WhitelistList 获取白名单
func (s *Service) WhitelistList(ctx context.Context) ([]pkgApisix.Route, error) {
	list, err := s.client.RouteWhitelistInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取白名单失败: %w", err)
	}
	return list, nil
}

// WhitelistUserDelete 移除白名单
func (s *Service) WhitelistUserDelete(ctx context.Context, routeID, consumerName string) error {
	if err := s.client.RouteWhitelistUserDelete(ctx, routeID, consumerName); err != nil {
		return fmt.Errorf("撤销白名单授权失败: %w", err)
	}
	return nil
}

// ─── Upstream 管理 ───

// UpstreamList 获取 Upstream 列表
func (s *Service) UpstreamList(ctx context.Context) ([]pkgApisix.Upstream, error) {
	list, err := s.client.UpstreamList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取上游列表失败: %w", err)
	}
	return list, nil
}

// UpstreamInspect 获取单条 Upstream 详情
func (s *Service) UpstreamInspect(ctx context.Context, upstreamID string) (*pkgApisix.Upstream, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	upstream, err := s.client.UpstreamInspect(ctx, upstreamID)
	if err != nil {
		return nil, fmt.Errorf("获取上游详情失败: %w", err)
	}
	return upstream, nil
}

// UpstreamCreate 创建 Upstream
func (s *Service) UpstreamCreate(ctx context.Context, req pkgApisix.Upstream) (*pkgApisix.Upstream, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !pkgApisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	upstream, err := s.client.UpstreamCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建上游失败: %w", err)
	}
	return upstream, nil
}

// UpstreamUpdate 更新 Upstream
func (s *Service) UpstreamUpdate(ctx context.Context, upstreamID string, req pkgApisix.Upstream) (*pkgApisix.Upstream, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !pkgApisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	upstream, err := s.client.UpstreamUpdate(ctx, upstreamID, req)
	if err != nil {
		return nil, fmt.Errorf("更新上游失败: %w", err)
	}
	return upstream, nil
}

// UpstreamDelete 删除 Upstream
func (s *Service) UpstreamDelete(ctx context.Context, upstreamID string) error {
	if upstreamID == "" {
		return fmt.Errorf("Upstream ID 不能为空")
	}
	if err := s.client.UpstreamDelete(ctx, upstreamID); err != nil {
		return fmt.Errorf("删除上游失败: %w", err)
	}
	return nil
}

// ─── SSL 证书管理 ───

// SSLList 获取 SSL 证书列表
func (s *Service) SSLList(ctx context.Context) ([]pkgApisix.SSL, error) {
	list, err := s.client.SSLList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取证书列表失败: %w", err)
	}
	return list, nil
}

// SSLInspect 获取单个 SSL 证书详情
func (s *Service) SSLInspect(ctx context.Context, sslID string) (*pkgApisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	ssl, err := s.client.SSLInspect(ctx, sslID)
	if err != nil {
		return nil, fmt.Errorf("获取证书详情失败: %w", err)
	}
	return ssl, nil
}

// SSLCreate 创建 SSL 证书
func (s *Service) SSLCreate(ctx context.Context, req pkgApisix.SSL) (*pkgApisix.SSL, error) {
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	if req.Cert == "" {
		return nil, fmt.Errorf("证书内容不能为空")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("私钥内容不能为空")
	}
	ssl, err := s.client.SSLCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建证书失败: %w", err)
	}
	return ssl, nil
}

// SSLUpdate 更新 SSL 证书
func (s *Service) SSLUpdate(ctx context.Context, sslID string, req pkgApisix.SSL) (*pkgApisix.SSL, error) {
	if sslID == "" {
		return nil, fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if len(req.Snis) == 0 {
		return nil, fmt.Errorf("SNI 不能为空")
	}
	ssl, err := s.client.SSLUpdate(ctx, sslID, req)
	if err != nil {
		return nil, fmt.Errorf("更新证书失败: %w", err)
	}
	return ssl, nil
}

// SSLDelete 删除 SSL 证书
func (s *Service) SSLDelete(ctx context.Context, sslID string) error {
	if sslID == "" {
		return fmt.Errorf("SSL 证书 ID 不能为空")
	}
	if err := s.client.SSLDelete(ctx, sslID); err != nil {
		return fmt.Errorf("删除证书失败: %w", err)
	}
	return nil
}

// ─── 辅助资源 ───

// PluginConfigList 获取 Plugin Config 列表
func (s *Service) PluginConfigList(ctx context.Context) ([]pkgApisix.PluginConfig, error) {
	list, err := s.client.PluginConfigList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取插件配置列表失败: %w", err)
	}
	return list, nil
}

// PluginConfigInspect 获取单个 Plugin Config 详情
func (s *Service) PluginConfigInspect(ctx context.Context, configID string) (*pkgApisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	config, err := s.client.PluginConfigInspect(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("获取插件配置详情失败: %w", err)
	}
	return config, nil
}

// PluginConfigCreate 创建 Plugin Config
func (s *Service) PluginConfigCreate(ctx context.Context, req pkgApisix.PluginConfig) (*pkgApisix.PluginConfig, error) {
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	req.ID = strutil.NewString()
	config, err := s.client.PluginConfigCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建插件配置失败: %w", err)
	}
	return config, nil
}

// PluginConfigUpdate 更新 Plugin Config
func (s *Service) PluginConfigUpdate(ctx context.Context, configID string, req pkgApisix.PluginConfig) (*pkgApisix.PluginConfig, error) {
	if configID == "" {
		return nil, fmt.Errorf("Plugin Config ID 不能为空")
	}
	if len(req.Plugins) == 0 {
		return nil, fmt.Errorf("插件配置不能为空")
	}
	config, err := s.client.PluginConfigUpdate(ctx, configID, req)
	if err != nil {
		return nil, fmt.Errorf("更新插件配置失败: %w", err)
	}
	return config, nil
}

// PluginConfigDelete 删除 Plugin Config
func (s *Service) PluginConfigDelete(ctx context.Context, configID string) error {
	if configID == "" {
		return fmt.Errorf("Plugin Config ID 不能为空")
	}
	if err := s.client.PluginConfigDelete(ctx, configID); err != nil {
		return fmt.Errorf("删除插件配置失败: %w", err)
	}
	return nil
}

// PluginList 获取可用插件列表
func (s *Service) PluginList(ctx context.Context) (any, error) {
	list, err := s.client.PluginList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}
	return list, nil
}
