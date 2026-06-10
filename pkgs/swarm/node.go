// Package swarm 提供 Swarm 相关功能
package swarm

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/libgo/logman"
)

// NodeList 获取节点列表，直接返回 Docker SDK 原始节点结构。
func (s *SwarmService) NodeList(ctx context.Context) ([]swarm.Node, error) {
	nodes, err := s.client.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		logman.Error("NodeList failed", "error", err)
		return nil, err
	}
	return nodes, nil
}

// NodeAction 节点操作（drain/active/pause/remove）
func (s *SwarmService) NodeAction(ctx context.Context, id, action string) error {
	if action == "remove" {
		if err := s.client.NodeRemove(ctx, id, swarm.NodeRemoveOptions{Force: true}); err != nil {
			logman.Error("NodeRemove failed", "id", id, "error", err)
			return err
		}
		return nil
	}

	node, _, err := s.client.NodeInspectWithRaw(ctx, id)
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

	if err := s.client.NodeUpdate(ctx, id, node.Version, node.Spec); err != nil {
		logman.Error("NodeUpdate failed", "error", err)
		return err
	}

	return nil
}

// NodeInspect 获取节点原始详情。
func (s *SwarmService) NodeInspect(ctx context.Context, id string) (swarm.Node, error) {
	node, _, err := s.client.NodeInspectWithRaw(ctx, id)
	if err != nil {
		logman.Error("NodeInspect failed", "id", id, "error", err)
		return swarm.Node{}, err
	}
	return node, nil
}
