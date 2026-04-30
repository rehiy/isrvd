package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/pango/logman"
)

// NodeDTO Swarm 节点信息
type NodeDTO struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Role          string `json:"role"`
	Availability  string `json:"availability"`
	State         string `json:"state"`
	Addr          string `json:"addr"`
	EngineVersion string `json:"engineVersion"`
	Leader        bool   `json:"leader"`
}

// ListNodes 获取节点列表
func (m *SwarmService) ListNodes(ctx context.Context) ([]NodeDTO, error) {
	nodes, err := m.client.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		logman.Error("NodeList failed", "error", err)
		return nil, err
	}

	var result []NodeDTO
	for _, n := range nodes {
		result = append(result, NodeDTO{
			ID:            n.ID,
			Hostname:      n.Description.Hostname,
			Role:          string(n.Spec.Role),
			Availability:  string(n.Spec.Availability),
			State:         string(n.Status.State),
			Addr:          n.Status.Addr,
			EngineVersion: n.Description.Engine.EngineVersion,
			Leader:        n.ManagerStatus != nil && n.ManagerStatus.Leader,
		})
	}

	return result, nil
}

// NodeAction 节点操作（drain/active/pause/remove）
func (m *SwarmService) NodeAction(ctx context.Context, id, action string) error {
	if action == "remove" {
		if err := m.client.NodeRemove(ctx, id, swarm.NodeRemoveOptions{Force: true}); err != nil {
			logman.Error("NodeRemove failed", "id", id, "error", err)
			return err
		}
		return nil
	}

	node, _, err := m.client.NodeInspectWithRaw(ctx, id)
	if err != nil {
		logman.Error("NodeInspect failed", "id", id, "error", err)
		return err
	}

	switch action {
	case "drain":
		node.Spec.Availability = swarm.NodeAvailabilityDrain
	case "active":
		node.Spec.Availability = swarm.NodeAvailabilityActive
	case "pause":
		node.Spec.Availability = swarm.NodeAvailabilityPause
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	if err := m.client.NodeUpdate(ctx, id, node.Version, node.Spec); err != nil {
		logman.Error("NodeUpdate failed", "error", err)
		return err
	}

	return nil
}

// NodeInspect 节点详情
type NodeInspect struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	Role          string            `json:"role"`
	Availability  string            `json:"availability"`
	State         string            `json:"state"`
	Addr          string            `json:"addr"`
	EngineVersion string            `json:"engineVersion"`
	Leader        bool              `json:"leader"`
	OS            string            `json:"os"`
	Architecture  string            `json:"architecture"`
	CPUs          int64             `json:"cpus"`
	MemoryBytes   int64             `json:"memoryBytes"`
	Labels        map[string]string `json:"labels"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

// InspectNode 获取节点详情
func (m *SwarmService) InspectNode(ctx context.Context, id string) (*NodeInspect, error) {
	node, _, err := m.client.NodeInspectWithRaw(ctx, id)
	if err != nil {
		logman.Error("NodeInspect failed", "id", id, "error", err)
		return nil, err
	}

	result := &NodeInspect{
		ID:            node.ID,
		Hostname:      node.Description.Hostname,
		Role:          string(node.Spec.Role),
		Availability:  string(node.Spec.Availability),
		State:         string(node.Status.State),
		Addr:          node.Status.Addr,
		EngineVersion: node.Description.Engine.EngineVersion,
		Leader:        node.ManagerStatus != nil && node.ManagerStatus.Leader,
		OS:            node.Description.Platform.OS,
		Architecture:  node.Description.Platform.Architecture,
		CPUs:          node.Description.Resources.NanoCPUs / 1e9,
		MemoryBytes:   node.Description.Resources.MemoryBytes,
		Labels:        node.Spec.Labels,
		CreatedAt:     node.Meta.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     node.Meta.UpdatedAt.Format(time.RFC3339),
	}

	return result, nil
}
