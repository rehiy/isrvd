// Package swarm 提供 Swarm 业务服务层
package swarm

import (
	"context"
	"fmt"

	"isrvd/internal/registry"
	pkgswarm "isrvd/pkgs/swarm"
)

// Service Swarm 业务服务
type Service struct {
	svc *pkgswarm.SwarmService
}

// NewService 创建 Swarm 业务服务，验证节点是否是 Swarm manager
func NewService() (*Service, error) {
	svc := registry.SwarmService
	if svc == nil {
		return nil, fmt.Errorf("Swarm 服务未初始化")
	}
	// 验证节点是否加入 Swarm 且为 manager
	if _, err := svc.GetClient().SwarmInspect(context.Background()); err != nil {
		return nil, fmt.Errorf("Swarm 不可用: %w", err)
	}
	return &Service{svc: svc}, nil
}

// CheckAvailability 检测 Swarm 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.svc == nil {
		return false
	}
	_, err := s.svc.GetClient().SwarmInspect(ctx)
	return err == nil
}

// Info 获取 Swarm 集群概览
func (s *Service) Info(ctx context.Context) (map[string]any, error) {
	info, err := s.svc.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Swarm 信息失败: %w", err)
	}
	return info, nil
}

// JoinToken 获取加入集群的 token
func (s *Service) JoinToken(ctx context.Context) (map[string]string, error) {
	tokens, err := s.svc.JoinToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取加入令牌失败: %w", err)
	}
	return tokens, nil
}

// NodeList 获取节点列表
func (s *Service) NodeList(ctx context.Context) ([]pkgswarm.NodeInfo, error) {
	list, err := s.svc.NodeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}
	return list, nil
}

// NodeAction 节点操作
func (s *Service) NodeAction(ctx context.Context, id, action string) error {
	if id == "" {
		return fmt.Errorf("节点 ID 不能为空")
	}
	if action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.svc.NodeAction(ctx, id, action); err != nil {
		return fmt.Errorf("节点操作 %s 失败: %w", action, err)
	}
	return nil
}

// NodeInspect 获取节点详情
func (s *Service) NodeInspect(ctx context.Context, id string) (*pkgswarm.NodeDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少节点 ID")
	}
	detail, err := s.svc.NodeInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取节点详情失败: %w", err)
	}
	return detail, nil
}

// ServiceList 获取服务列表
func (s *Service) ServiceList(ctx context.Context) ([]pkgswarm.ServiceInfo, error) {
	list, err := s.svc.ServiceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}
	return list, nil
}

// ServiceInspect 获取服务详情
func (s *Service) ServiceInspect(ctx context.Context, id string) (*pkgswarm.ServiceDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	detail, err := s.svc.ServiceInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取服务详情失败: %w", err)
	}
	return detail, nil
}

// ServiceCreate 创建服务
func (s *Service) ServiceCreate(ctx context.Context, req pkgswarm.ServiceSpec) (string, error) {
	if req.Name == "" {
		return "", fmt.Errorf("服务名称不能为空")
	}
	if req.Image == "" {
		return "", fmt.Errorf("镜像名称不能为空")
	}
	id, err := s.svc.ServiceCreate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("创建服务失败: %w", err)
	}
	return id, nil
}

// ServiceAction 服务操作
func (s *Service) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	if id == "" {
		return fmt.Errorf("服务 ID 不能为空")
	}
	if action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.svc.ServiceAction(ctx, id, action, replicas); err != nil {
		return fmt.Errorf("服务操作 %s 失败: %w", action, err)
	}
	return nil
}

// ServiceForceUpdate 强制重新部署服务
func (s *Service) ServiceForceUpdate(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("服务 ID 不能为空")
	}
	if err := s.svc.ServiceForceUpdate(ctx, id); err != nil {
		return fmt.Errorf("强制重新部署服务失败: %w", err)
	}
	return nil
}

// ServiceLogs 获取服务日志
func (s *Service) ServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	logs, err := s.svc.ServiceLogs(ctx, serviceID, tail)
	if err != nil {
		return nil, fmt.Errorf("获取服务日志失败: %w", err)
	}
	return logs, nil
}

// TaskList 获取任务列表
func (s *Service) TaskList(ctx context.Context, serviceID string) ([]pkgswarm.Task, error) {
	list, err := s.svc.TaskList(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}
	return list, nil
}
