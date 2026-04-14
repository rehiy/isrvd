package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rehiy/pango/logman"
)

// ListNetworks 列出网络
func (s *DockerService) ListNetworks(ctx context.Context) ([]*NetworkInfo, error) {
	networks, err := s.client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		logman.Error("List networks failed", "error", err)
		return nil, err
	}

	var result []*NetworkInfo
	for _, net := range networks {
		subnet := ""
		if len(net.IPAM.Config) > 0 && net.IPAM.Config[0].Subnet != "" {
			subnet = net.IPAM.Config[0].Subnet
		}
		id := net.ID
		if len(id) > 12 {
			id = id[:12]
		}
		result = append(result, &NetworkInfo{
			ID: id, Name: net.Name, Driver: net.Driver, Subnet: subnet, Scope: net.Scope,
		})
	}

	return result, nil
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

// CreateNetwork 创建网络
func (s *DockerService) CreateNetwork(ctx context.Context, name, driver string) (string, error) {
	if driver == "" {
		driver = "bridge"
	}

	resp, err := s.client.NetworkCreate(ctx, name, types.NetworkCreate{Driver: driver})
	if err != nil {
		logman.Error("Create network failed", "error", err)
		return "", err
	}

	id := resp.ID
	if len(id) > 12 {
		id = id[:12]
	}

	logman.Info("Network created", "name", name, "id", id)
	return id, nil
}

// InspectNetwork 获取网络详情
func (s *DockerService) InspectNetwork(ctx context.Context, id string) (*NetworkInspectResponse, error) {
	networkInfo, err := s.client.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		logman.Error("Network inspect failed", "id", id, "error", err)
		return nil, err
	}

	// 提取已连接的容器信息
	var containers []*NetworkContainerInfo
	for endpointID, ep := range networkInfo.Containers {
		name := ep.Name
		if name == "" {
			name = endpointID[:12]
		}
		containerJSON, err := s.client.ContainerInspect(ctx, ep.Name)
		if err == nil && len(containerJSON.Name) > 0 {
			name = strings.TrimPrefix(containerJSON.Name, "/")
		}
		containers = append(containers, &NetworkContainerInfo{
			ID:         endpointID[:12],
			Name:       name,
			IPv4:       ep.IPv4Address,
			IPv6:       ep.IPv6Address,
			MacAddress: ep.MacAddress,
		})
	}

	result := &NetworkInspectResponse{
		ID:         networkInfo.ID,
		Name:       networkInfo.Name,
		Driver:     networkInfo.Driver,
		Scope:      networkInfo.Scope,
		Internal:   networkInfo.Internal,
		EnableIPv6: networkInfo.EnableIPv6,
		Containers: containers,
	}

	if len(networkInfo.IPAM.Config) > 0 {
		result.Subnet = networkInfo.IPAM.Config[0].Subnet
		result.Gateway = networkInfo.IPAM.Config[0].Gateway
	}

	return result, nil
}
