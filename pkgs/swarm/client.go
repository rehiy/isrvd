package swarm

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"
)

// SwarmService Swarm 业务逻辑服务
type SwarmService struct {
	client *client.Client
}

// NewSwarmService 创建 Swarm 服务
func NewSwarmService(dockerClient *client.Client) *SwarmService {
	return &SwarmService{client: dockerClient}
}

// GetClient 获取 Docker 客户端
func (m *SwarmService) GetClient() *client.Client {
	return m.client
}

// GetJoinTokens 获取加入集群的 token
func (m *SwarmService) GetJoinTokens(ctx context.Context) (map[string]string, error) {
	info, err := m.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}
	return map[string]string{
		"worker":  info.JoinTokens.Worker,
		"manager": info.JoinTokens.Manager,
	}, nil
}

// GetSwarmInfo 获取 Swarm 集群概览
func (m *SwarmService) GetSwarmInfo(ctx context.Context) (map[string]any, error) {
	info, err := m.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}

	nodes, _ := m.client.NodeList(ctx, swarm.NodeListOptions{})
	services, _ := m.client.ServiceList(ctx, swarm.ServiceListOptions{})
	tasks, _ := m.client.TaskList(ctx, swarm.TaskListOptions{})

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
