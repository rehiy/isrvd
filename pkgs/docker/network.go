package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/rehiy/libgo/logman"
)

// NetworkList 列出网络，直接返回 Docker SDK 原始列表项。
func (s *DockerService) NetworkList(ctx context.Context) ([]network.Summary, error) {
	networks, err := s.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		logman.Error("List networks failed", "error", err)
		return nil, err
	}
	return networks, nil
}

// NetworkAction 网络操作
func (s *DockerService) NetworkAction(ctx context.Context, id, action string) error {
	switch action {
	case "remove":
		if err := s.client.NetworkRemove(ctx, id); err != nil {
			logman.Error("Remove network failed", "id", id, "error", err)
			return err
		}
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	logman.Info("Network action performed", "action", action, "id", id)
	return nil
}

// NetworkCreate 创建网络
func (s *DockerService) NetworkCreate(ctx context.Context, name, driver, subnet string) (string, error) {
	if driver == "" {
		driver = "bridge"
	}

	options := network.CreateOptions{Driver: driver}
	if subnet != "" {
		options.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{{Subnet: subnet}},
		}
	}

	resp, err := s.client.NetworkCreate(ctx, name, options)
	if err != nil {
		logman.Error("Create network failed", "error", err)
		return "", err
	}

	logman.Info("Network created", "name", name, "id", ShortID(resp.ID))
	return ShortID(resp.ID), nil
}

// NetworkInspect 获取网络原始详情。
func (s *DockerService) NetworkInspect(ctx context.Context, id string) (network.Inspect, error) {
	networkInfo, err := s.client.NetworkInspect(ctx, id, network.InspectOptions{})
	if err != nil {
		logman.Error("Network inspect failed", "id", id, "error", err)
		return network.Inspect{}, err
	}
	return networkInfo, nil
}
