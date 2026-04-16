package swarm

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"
)

// SwarmManager Swarm 业务逻辑管理器
type SwarmManager struct {
	client *client.Client
}

// NewSwarmManager 创建 Swarm 管理器
func NewSwarmManager(dockerClient *client.Client) *SwarmManager {
	return &SwarmManager{client: dockerClient}
}

// GetClient 获取 Docker 客户端
func (m *SwarmManager) GetClient() *client.Client {
	return m.client
}

// GetSwarmInfo 获取 Swarm 集群概览
func (m *SwarmManager) GetSwarmInfo(ctx context.Context) (map[string]interface{}, error) {
	info, err := m.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}

	nodes, _ := m.client.NodeList(ctx, types.NodeListOptions{})
	services, _ := m.client.ServiceList(ctx, types.ServiceListOptions{})
	tasks, _ := m.client.TaskList(ctx, types.TaskListOptions{})

	var managers, workers int
	for _, n := range nodes {
		if n.Spec.Role == swarm.NodeRoleManager {
			managers++
		} else {
			workers++
		}
	}

	return map[string]interface{}{
		"clusterID": info.ID,
		"createdAt": info.Meta.CreatedAt.Format(time.RFC3339),
		"nodes":     len(nodes),
		"managers":  managers,
		"workers":   workers,
		"services":  len(services),
		"tasks":     len(tasks),
	}, nil
}
