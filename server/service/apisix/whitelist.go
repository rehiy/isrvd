package apisix

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"isrvd/pkgs/apisix"
)

// WhitelistCreateRequest 配置访问授权路由请求
type WhitelistCreateRequest struct {
	RouteID   string         `json:"route_id"`  // 路由 ID
	Consumers []string       `json:"consumers"` // 授权 Consumer 列表
	KeyAuth   map[string]any `json:"key_auth"`  // Key Auth 插件配置
}

// inspectRouteAuth 读取路由并分析认证来源；授权来自 PluginConfig 的路由不在路由上修改，返回 ErrACLInherited
func (s *Service) inspectRouteAuth(ctx context.Context, routeID string) (*apisix.Route, *apisix.RouteAuthInfo, error) {
	route, err := s.client.RouteInspect(ctx, routeID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取路由详情失败: %w", err)
	}
	info, err := s.client.RouteAuthInspect(ctx, route)
	if err != nil {
		return nil, nil, fmt.Errorf("获取插件配置详情失败: %w", err)
	}
	if info.ACLInherited {
		return nil, nil, apisix.ErrACLInherited
	}
	return route, info, nil
}

// requireKeyAuth 校验 Consumer 已配置 key-auth；认证由别处管理时所需认证方式未知，不做限制
func requireKeyAuth(c apisix.Consumer, info *apisix.RouteAuthInfo) error {
	if _, ok := c.Plugins["key-auth"]; !ok && !info.ManagedElsewhere {
		return fmt.Errorf("消费者 %s 未配置 key-auth", c.Username)
	}
	return nil
}

// validateKeyAuth 校验写入路由的 key-auth 配置；认证由别处管理时不会写入，无需校验
func validateKeyAuth(keyAuth map[string]any, info *apisix.RouteAuthInfo) error {
	if info.ManagedElsewhere {
		return nil
	}
	if len(keyAuth) == 0 {
		return fmt.Errorf("key-auth 配置不能为空")
	}
	header, _ := keyAuth["header"].(string)
	if strings.TrimSpace(header) == "" {
		return fmt.Errorf("key-auth 请求头名称不能为空")
	}
	return nil
}

// WhitelistCreate 为已有路由配置 Consumer 访问授权。
// 仅当 consumers 显式传入空数组时视为删除授权（移除 consumer-restriction 和 key-auth 插件）；
// 字段缺失或为 null 视为参数错误，避免漏传字段时误将路由变为公开。
func (s *Service) WhitelistCreate(ctx context.Context, req WhitelistCreateRequest) (*apisix.Route, error) {
	routeID := strings.TrimSpace(req.RouteID)
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}
	if req.Consumers == nil {
		return nil, fmt.Errorf("授权用户不能为空")
	}

	route, info, err := s.inspectRouteAuth(ctx, routeID)
	if err != nil {
		return nil, err
	}

	// 编辑场景：consumers 为空数组 → 删除授权
	if len(req.Consumers) == 0 {
		if err := s.client.RouteConsumerRestrictionUpdate(ctx, route, info, nil, nil); err != nil {
			return nil, fmt.Errorf("删除访问授权失败: %w", err)
		}
		return s.RouteInspect(ctx, routeID)
	}
	if err := validateKeyAuth(req.KeyAuth, info); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("授权用户不能为空")
	}

	existingConsumers, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	existingConsumerMap := make(map[string]apisix.Consumer, len(existingConsumers))
	for _, consumer := range existingConsumers {
		existingConsumerMap[consumer.Username] = consumer
	}
	for _, name := range consumers {
		consumer, ok := existingConsumerMap[name]
		if !ok {
			return nil, fmt.Errorf("Consumer %s 不存在，请先创建该 Consumer", name)
		}
		if err := requireKeyAuth(consumer, info); err != nil {
			return nil, err
		}
	}
	if err := s.client.RouteConsumerRestrictionUpdate(ctx, route, info, consumers, req.KeyAuth); err != nil {
		return nil, fmt.Errorf("配置访问授权失败: %w", err)
	}
	return s.RouteInspect(ctx, routeID)
}

// WhitelistUserCreateRequest 新建用户并加入访问授权请求
type WhitelistUserCreateRequest struct {
	RouteID  string         `json:"route_id"` // 目标路由 ID
	Username string         `json:"username"` // Consumer 用户名
	Key      string         `json:"key"`      // key-auth 密钥
	KeyAuth  map[string]any `json:"key_auth"` // key-auth 插件的附加配置
}

// WhitelistUserCreate 依次创建 Consumer（已存在则复用）并追加到路由白名单，非原子；
// 追加失败时回滚本次新建的 Consumer。已存在的 Consumer 不会更新其 key；新建的 Consumer 只配置 key-auth。
func (s *Service) WhitelistUserCreate(ctx context.Context, req WhitelistUserCreateRequest) (*apisix.Route, error) {
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

	// 先确认路由可用，避免留下孤立的 Consumer
	route, info, err := s.inspectRouteAuth(ctx, routeID)
	if err != nil {
		return nil, err
	}
	if err := validateKeyAuth(req.KeyAuth, info); err != nil {
		return nil, err
	}
	consumers := apisix.RouteWhitelist(route)
	if slices.Contains(consumers, username) {
		// 已在授权列表中，直接返回当前路由
		return route, nil
	}

	// 步骤一：创建 Consumer，若已存在则复用
	existingConsumers, err := s.client.ConsumerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取消费者列表失败: %w", err)
	}
	idx := slices.IndexFunc(existingConsumers, func(c apisix.Consumer) bool { return c.Username == username })
	exists := idx >= 0
	if exists {
		if err := requireKeyAuth(existingConsumers[idx], info); err != nil {
			return nil, err
		}
	} else {
		plugins := map[string]any{
			"key-auth": map[string]any{"key": key},
		}
		if _, err := s.client.ConsumerCreate(ctx, username, "", plugins); err != nil {
			return nil, fmt.Errorf("创建消费者失败: %w", err)
		}
	}

	// 步骤二：将 Consumer 加入路由访问授权，失败时回滚本次新建的 Consumer
	consumers = append(consumers, username)
	if err := s.client.RouteConsumerRestrictionUpdate(ctx, route, info, consumers, req.KeyAuth); err != nil {
		if !exists {
			_ = s.client.ConsumerDelete(ctx, username)
		}
		return nil, fmt.Errorf("配置访问授权失败: %w", err)
	}
	return s.RouteInspect(ctx, routeID)
}

// WhitelistList 获取访问授权
func (s *Service) WhitelistList(ctx context.Context) ([]apisix.Route, error) {
	list, err := s.client.RouteWhitelistInspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取访问授权失败: %w", err)
	}
	return list, nil
}
