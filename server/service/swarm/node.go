package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// NodeInfo Swarm 节点信息（列表项），保持前端稳定响应结构。
type NodeInfo struct {
	ID            string `json:"id"`            // 节点 ID
	Hostname      string `json:"hostname"`      // 主机名
	Role          string `json:"role"`          // 角色（manager/worker）
	Availability  string `json:"availability"`  // 可用状态（active/pause/drain）
	State         string `json:"state"`         // 节点状态（ready/down）
	Addr          string `json:"addr"`          // 节点地址
	EngineVersion string `json:"engineVersion"` // Docker 引擎版本
	Leader        bool   `json:"leader"`        // 是否为 Leader 节点
}

// NodeList 获取节点列表
func (s *Service) NodeList(ctx context.Context) ([]NodeInfo, error) {
	list, err := s.svc.NodeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}
	result := make([]NodeInfo, 0, len(list))
	for _, node := range list {
		result = append(result, nodeInfoFromRaw(node))
	}
	return result, nil
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

// NodeDetail 节点详情，保持前端稳定响应结构。
type NodeDetail struct {
	ID            string            `json:"id"`            // 节点 ID
	Hostname      string            `json:"hostname"`      // 主机名
	Role          string            `json:"role"`          // 角色（manager/worker）
	Availability  string            `json:"availability"`  // 可用状态
	State         string            `json:"state"`         // 节点状态
	Addr          string            `json:"addr"`          // 节点地址
	EngineVersion string            `json:"engineVersion"` // Docker 引擎版本
	Leader        bool              `json:"leader"`        // 是否为 Leader
	OS            string            `json:"os"`            // 操作系统
	Architecture  string            `json:"architecture"`  // CPU 架构
	CPUs          int64             `json:"cpus"`          // CPU 核心数
	MemoryBytes   int64             `json:"memoryBytes"`   // 内存大小（字节）
	Labels        map[string]string `json:"labels"`        // 节点标签
	CreatedAt     string            `json:"createdAt"`     // 创建时间
	UpdatedAt     string            `json:"updatedAt"`     // 更新时间
}

// NodeInspect 获取节点详情
func (s *Service) NodeInspect(ctx context.Context, id string) (*NodeDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少节点 ID")
	}
	node, err := s.svc.NodeInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取节点详情失败: %w", err)
	}
	return nodeDetailFromRaw(node), nil
}

func nodeInfoFromRaw(node swarm.Node) NodeInfo {
	return NodeInfo{
		ID:            node.ID,
		Hostname:      node.Description.Hostname,
		Role:          string(node.Spec.Role),
		Availability:  string(node.Spec.Availability),
		State:         string(node.Status.State),
		Addr:          node.Status.Addr,
		EngineVersion: node.Description.Engine.EngineVersion,
		Leader:        node.ManagerStatus != nil && node.ManagerStatus.Leader,
	}
}

func nodeDetailFromRaw(node swarm.Node) *NodeDetail {
	info := nodeInfoFromRaw(node)
	return &NodeDetail{
		ID: info.ID, Hostname: info.Hostname, Role: info.Role, Availability: info.Availability, State: info.State,
		Addr: info.Addr, EngineVersion: info.EngineVersion, Leader: info.Leader,
		OS: node.Description.Platform.OS, Architecture: node.Description.Platform.Architecture,
		CPUs: node.Description.Resources.NanoCPUs / 1e9, MemoryBytes: node.Description.Resources.MemoryBytes,
		Labels: node.Spec.Labels, CreatedAt: node.Meta.CreatedAt.Format(time.RFC3339), UpdatedAt: node.Meta.UpdatedAt.Format(time.RFC3339),
	}
}
