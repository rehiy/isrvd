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

// NewService 创建 Swarm 业务服务
func NewService() *Service {
	return &Service{svc: registry.SwarmService}
}

// CheckAvailability 检测 Swarm 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.svc == nil {
		return false
	}
	_, err := s.svc.GetClient().SwarmInspect(ctx)
	return err == nil
}

// SwarmInfo 获取 Swarm 集群概览
func (s *Service) SwarmInfo(ctx context.Context) (any, error) {
	return s.svc.GetSwarmInfo(ctx)
}

// GetJoinTokens 获取加入集群的 token
func (s *Service) GetJoinTokens(ctx context.Context) (any, error) {
	return s.svc.GetJoinTokens(ctx)
}

// ListNodes 获取节点列表
func (s *Service) ListNodes(ctx context.Context) (any, error) {
	return s.svc.ListNodes(ctx)
}

// NodeAction 节点操作
func (s *Service) NodeAction(ctx context.Context, id, action string) error {
	return s.svc.NodeAction(ctx, id, action)
}

// InspectNode 获取节点详情
func (s *Service) InspectNode(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少节点 ID")
	}
	return s.svc.InspectNode(ctx, id)
}

// ListServices 获取服务列表
func (s *Service) ListServices(ctx context.Context) (any, error) {
	return s.svc.ListServices(ctx)
}

// InspectService 获取服务详情
func (s *Service) InspectService(ctx context.Context, id string) (*pkgswarm.ServiceInfo, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	return s.svc.InspectService(ctx, id)
}

// CreateService 创建服务
func (s *Service) CreateService(ctx context.Context, req pkgswarm.ServiceSpec) (string, error) {
	return s.svc.CreateService(ctx, req)
}

// ServiceAction 服务操作
func (s *Service) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	return s.svc.ServiceAction(ctx, id, action, replicas)
}

// ForceUpdateService 强制重新部署服务
func (s *Service) ForceUpdateService(ctx context.Context, id string) error {
	return s.svc.ForceUpdateService(ctx, id)
}

// GetServiceLogs 获取服务日志
func (s *Service) GetServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	return s.svc.GetServiceLogs(ctx, serviceID, tail)
}

// ListTasks 获取任务列表
func (s *Service) ListTasks(ctx context.Context, serviceID string) (any, error) {
	return s.svc.ListTasks(ctx, serviceID)
}
