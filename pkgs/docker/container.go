package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/rehiy/libgo/logman"
)

// ContainerList 获取容器列表，直接返回 Docker SDK 原始列表项。
func (s *DockerService) ContainerList(ctx context.Context, all bool) ([]container.Summary, error) {
	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		logman.Error("List containers failed", "error", err)
		return nil, err
	}
	return containers, nil
}

// ContainerListByLabel 按 Docker label 查询并返回容器原始配置。
func (s *DockerService) ContainerListByLabel(ctx context.Context, key, value string) ([]container.InspectResponse, error) {
	label := key
	if value != "" {
		label += "=" + value
	}
	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", label)),
	})
	if err != nil {
		logman.Error("List containers by label failed", "label", label, "error", err)
		return nil, err
	}

	result := make([]container.InspectResponse, 0, len(containers))
	for _, ct := range containers {
		info, err := s.ContainerInspect(ctx, ct.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

// ContainerInspect 获取容器原始配置。
func (s *DockerService) ContainerInspect(ctx context.Context, id string) (container.InspectResponse, error) {
	info, err := s.client.ContainerInspect(ctx, id)
	if err != nil {
		logman.Error("Inspect container failed", "id", id, "error", err)
		return container.InspectResponse{}, err
	}
	return info, nil
}

// ContainerInspectRaw 保留旧调用名，返回 Docker SDK 原始类型。
func (s *DockerService) ContainerInspectRaw(ctx context.Context, id string) (container.InspectResponse, error) {
	return s.ContainerInspect(ctx, id)
}

// ContainerAction 容器操作（start/stop/restart/remove/pause/unpause）
func (s *DockerService) ContainerAction(ctx context.Context, id, action string) error {
	switch action {
	case "stop", "restart", "remove", "pause":
		selfID := s.GetSelfContainerID(ctx)
		if selfID != "" && (id == selfID || ShortID(id) == ShortID(selfID)) {
			return fmt.Errorf("禁止操作当前 iSrvd 所在容器")
		}
	}

	var err error
	switch action {
	case "start":
		err = s.client.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		timeout := 10
		err = s.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
	case "restart":
		timeout := 10
		err = s.client.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
	case "remove":
		err = s.client.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	case "pause":
		err = s.client.ContainerPause(ctx, id)
	case "unpause":
		err = s.client.ContainerUnpause(ctx, id)
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	if err != nil {
		logman.Error("Container action failed", "action", action, "id", id, "error", err)
		return err
	}

	logman.Info("Container action performed", "action", action, "id", id)
	return nil
}

// ContainerCreate 创建容器，直接接收 Docker SDK 原始配置。
func (s *DockerService) ContainerCreate(ctx context.Context, name string, containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) (string, error) {
	if containerConfig == nil {
		return "", fmt.Errorf("container config 不能为空")
	}
	if hostConfig == nil {
		hostConfig = &container.HostConfig{}
	}

	resp, err := s.client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, name)
	if err != nil {
		logman.Error("Create container failed", "name", name, "error", err)
		return "", err
	}

	logman.Info("Container created", "id", ShortID(resp.ID), "name", name)
	return resp.ID, nil
}

// ContainerStart 启动容器。
func (s *DockerService) ContainerStart(ctx context.Context, id string) error {
	if err := s.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		logman.Error("Start container failed", "id", ShortID(id), "error", err)
		return err
	}
	return nil
}

// ContainerCreateAndStart 创建并启动容器；启动失败时自动移除已创建容器。
func (s *DockerService) ContainerCreateAndStart(ctx context.Context, name string, containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) (string, error) {
	id, err := s.ContainerCreate(ctx, name, containerConfig, hostConfig, networkingConfig)
	if err != nil {
		return "", err
	}
	if err := s.ContainerStart(ctx, id); err != nil {
		if rmErr := s.client.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}); rmErr != nil {
			logman.Warn("Remove container after start failure", "id", ShortID(id), "error", rmErr)
		}
		return "", fmt.Errorf("启动容器失败: %w", err)
	}
	return id, nil
}
