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
	return s.svc.Info(ctx)
}

// JoinToken 获取加入集群的 token
func (s *Service) JoinToken(ctx context.Context) (map[string]string, error) {
	return s.svc.JoinToken(ctx)
}

// NodeList 获取节点列表
func (s *Service) NodeList(ctx context.Context) ([]pkgswarm.NodeInfo, error) {
	return s.svc.NodeList(ctx)
}

// NodeAction 节点操作
func (s *Service) NodeAction(ctx context.Context, id, action string) error {
	return s.svc.NodeAction(ctx, id, action)
}

// NodeInspect 获取节点详情
func (s *Service) NodeInspect(ctx context.Context, id string) (*pkgswarm.NodeDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少节点 ID")
	}
	return s.svc.NodeInspect(ctx, id)
}

// ServiceList 获取服务列表
func (s *Service) ServiceList(ctx context.Context) ([]pkgswarm.ServiceInfo, error) {
	return s.svc.ServiceList(ctx)
}

// ServiceInspect 获取服务详情
func (s *Service) ServiceInspect(ctx context.Context, id string) (*pkgswarm.ServiceDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	return s.svc.ServiceInspect(ctx, id)
}

// ServiceCreate 创建服务
func (s *Service) ServiceCreate(ctx context.Context, req pkgswarm.ServiceSpec) (string, error) {
	return s.svc.ServiceCreate(ctx, req)
}

// ServiceAction 服务操作
func (s *Service) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	return s.svc.ServiceAction(ctx, id, action, replicas)
}

// ServiceForceUpdate 强制重新部署服务
func (s *Service) ServiceForceUpdate(ctx context.Context, id string) error {
	return s.svc.ServiceForceUpdate(ctx, id)
}

// ServiceLogs 获取服务日志
func (s *Service) ServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	return s.svc.ServiceLogs(ctx, serviceID, tail)
}

// TaskList 获取任务列表
func (s *Service) TaskList(ctx context.Context, serviceID string) ([]pkgswarm.Task, error) {
	return s.svc.TaskList(ctx, serviceID)
}
