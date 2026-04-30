package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/network"
	"github.com/rehiy/pango/logman"
)

// NetworkInfo Docker 网络信息
type NetworkInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Driver  string   `json:"driver"`
	Subnet  string   `json:"subnet"`
	Scope   string   `json:"scope"`
	Network []string `json:"networks,omitempty"`
}

// ListNetworks 列出网络
func (s *DockerService) ListNetworks(ctx context.Context) ([]*NetworkInfo, error) {
	networks, err := s.client.NetworkList(ctx, network.ListOptions{})
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

// NetworkActionRequest 网络操作请求
type NetworkActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
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

// NetworkCreateRequest 创建网络请求
type NetworkCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

// CreateNetwork 创建网络
func (s *DockerService) CreateNetwork(ctx context.Context, name, driver string) (string, error) {
	if driver == "" {
		driver = "bridge"
	}

	resp, err := s.client.NetworkCreate(ctx, name, network.CreateOptions{Driver: driver})
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

// NetworkContainerInfo 网络中的容器信息
type NetworkContainerInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
	MacAddress string `json:"macAddress"`
}

// NetworkInspectResponse 网络详情响应
type NetworkInspectResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Driver     string                  `json:"driver"`
	Scope      string                  `json:"scope"`
	Subnet     string                  `json:"subnet"`
	Gateway    string                  `json:"gateway"`
	Internal   bool                    `json:"internal"`
	EnableIPv6 bool                    `json:"enableIPv6"`
	Containers []*NetworkContainerInfo `json:"containers"`
}

// InspectNetwork 获取网络详情
func (s *DockerService) InspectNetwork(ctx context.Context, id string) (*NetworkInspectResponse, error) {
	networkInfo, err := s.client.NetworkInspect(ctx, id, network.InspectOptions{})
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
