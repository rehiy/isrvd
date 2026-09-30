package apisix

import (
	"context"
	"fmt"

	"isrvd/pkgs/apisix"
)

// UpstreamList 获取 Upstream 列表
func (s *Service) UpstreamList(ctx context.Context) ([]apisix.Upstream, error) {
	list, err := s.client.UpstreamList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取上游列表失败: %w", err)
	}
	return list, nil
}

// UpstreamInspect 获取单条 Upstream 详情
func (s *Service) UpstreamInspect(ctx context.Context, upstreamID string) (*apisix.Upstream, error) {
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
func (s *Service) UpstreamCreate(ctx context.Context, req apisix.Upstream) (*apisix.Upstream, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !apisix.HasUpstreamNodes(req.Nodes) {
		return nil, fmt.Errorf("Upstream 节点不能为空")
	}
	upstream, err := s.client.UpstreamCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建上游失败: %w", err)
	}
	return upstream, nil
}

// UpstreamUpdate 更新 Upstream
func (s *Service) UpstreamUpdate(ctx context.Context, upstreamID string, req apisix.Upstream) (*apisix.Upstream, error) {
	if upstreamID == "" {
		return nil, fmt.Errorf("Upstream ID 不能为空")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("Upstream 名称不能为空")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("Upstream 类型不能为空")
	}
	if !apisix.HasUpstreamNodes(req.Nodes) {
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
