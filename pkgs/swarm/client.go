package swarm

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/rehiy/libgo/logman"
)

// SwarmService Swarm 业务逻辑服务
type SwarmService struct {
	client *client.Client
}

// NewSwarmService 创建 Swarm 服务
func NewSwarmService(dockerClient *client.Client) *SwarmService {
	return &SwarmService{client: dockerClient}
}

// Client 获取 Docker 客户端
func (s *SwarmService) Client() *client.Client {
	return s.client
}

// JoinToken 获取加入集群的 token
func (s *SwarmService) JoinToken(ctx context.Context) (map[string]string, error) {
	info, err := s.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}
	return map[string]string{
		"worker":  info.JoinTokens.Worker,
		"manager": info.JoinTokens.Manager,
	}, nil
}

// Info 获取 Swarm 集群概览
func (s *SwarmService) Info(ctx context.Context) (map[string]any, error) {
	info, err := s.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}

	nodes, err := s.client.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		logman.Warn("NodeList failed in Info", "error", err)
	}
	services, err := s.client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		logman.Warn("ServiceList failed in Info", "error", err)
	}
	tasks, err := s.client.TaskList(ctx, swarm.TaskListOptions{})
	if err != nil {
		logman.Warn("TaskList failed in Info", "error", err)
	}

	var managers, workers int
	for _, n := range nodes {
		if n.Spec.Role == swarm.NodeRoleManager {
			managers++
		} else {
			workers++
		}
	}

	return map[string]any{
		"clusterID": info.ID,
		"createdAt": info.Meta.CreatedAt.Format(time.RFC3339),
		"nodes":     len(nodes),
		"managers":  managers,
		"workers":   workers,
		"services":  len(services),
		"tasks":     len(tasks),
	}, nil
}
