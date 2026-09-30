package apisix

import (
	"context"
	"fmt"

	"isrvd/pkgs/apisix"
)

// RouteList 获取所有路由列表
func (s *Service) RouteList(ctx context.Context) ([]apisix.Route, error) {
	list, err := s.client.RouteList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取路由列表失败: %w", err)
	}
	return list, nil
}

// RouteInspect 获取单条路由详情
func (s *Service) RouteInspect(ctx context.Context, routeID string) (*apisix.Route, error) {
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
func (s *Service) RouteCreate(ctx context.Context, req apisix.Route) (*apisix.Route, error) {
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
func (s *Service) RouteUpdate(ctx context.Context, routeID string, req apisix.Route) (*apisix.Route, error) {
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
