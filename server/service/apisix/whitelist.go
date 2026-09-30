package apisix

import (
	"context"
	"fmt"
	"strings"

	"isrvd/pkgs/apisix"
)

// WhitelistCreateRequest 配置访问授权路由请求
type WhitelistCreateRequest struct {
	RouteID   string         `json:"route_id"`  // 路由 ID
	Consumers []string       `json:"consumers"` // 授权 Consumer 列表
	KeyAuth   map[string]any `json:"key_auth"`  // Key Auth 插件配置
}

// WhitelistCreate 为已有路由配置 Consumer 访问授权。
// 编辑场景下 consumers 为空时，视为删除授权（移除 consumer-restriction 和 key-auth 插件）。
func (s *Service) WhitelistCreate(ctx context.Context, req WhitelistCreateRequest) (*apisix.Route, error) {
	routeID := strings.TrimSpace(req.RouteID)
	if routeID == "" {
		return nil, fmt.Errorf("路由 ID 不能为空")
	}

	// 编辑场景：consumers 为空 → 删除授权
	if len(req.Consumers) == 0 {
		if err := s.client.RouteConsumerRestrictionUpdate(ctx, routeID, nil, nil); err != nil {
			return nil, fmt.Errorf("删除访问授权失败: %w", err)
		}
		return s.RouteInspect(ctx, routeID)
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
		return nil, fmt.Errorf("授权用户不能为空")
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

// WhitelistUserCreate 原子操作：创建 Consumer（含 key-auth）并加入路由访问授权。
// 若 Consumer 已存在则直接复用，不报错，保证接口幂等。
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

	// 步骤二：将 Consumer 加入路由访问授权
	route, err := s.client.RouteInspect(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("获取路由详情失败: %w", err)
	}
	consumers := route.Consumers
	for _, c := range consumers {
		if c == username {
			// 已在授权列表中，直接返回当前路由
			return route, nil
		}
	}
	consumers = append(consumers, username)
	if err := s.client.RouteConsumerRestrictionUpdate(ctx, routeID, consumers, req.KeyAuth); err != nil {
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
