package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/rehiy/libgo/logman"
)

// VolumeMapping 挂载映射
type VolumeMapping struct {
	Type          string `json:"type,omitempty"`
	Source        string `json:"source,omitempty"`
	HostPath      string `json:"hostPath,omitempty"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

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

// ContainerCreate 创建并启动容器，直接接收 Docker SDK 原始配置。
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

	if err := s.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		logman.Error("Start container failed", "id", ShortID(resp.ID), "name", name, "error", err)
		if rmErr := s.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); rmErr != nil {
			logman.Warn("Remove container after start failure", "id", ShortID(resp.ID), "error", rmErr)
		}
		return "", fmt.Errorf("启动容器失败: %w", err)
	}

	logman.Info("Container created", "id", ShortID(resp.ID), "name", name)
	return resp.ID, nil
}

func (s *DockerService) buildMount(containerName string, vol VolumeMapping) (mount.Mount, error) {
	mountType := strings.ToLower(strings.TrimSpace(vol.Type))
	source := firstNonEmpty(vol.Source, vol.HostPath)
	if source == "" {
		return mount.Mount{}, fmt.Errorf("挂载源不能为空")
	}
	if vol.ContainerPath == "" {
		return mount.Mount{}, fmt.Errorf("挂载目标不能为空")
	}
	if mountType == "" {
		mountType = inferMountType(source)
	}

	switch mountType {
	case string(mount.TypeVolume):
		// volume 名不能含路径分隔符
		if strings.ContainsRune(source, '/') {
			return mount.Mount{}, fmt.Errorf("volume 名称不能包含路径分隔符，请使用 bind 类型或改用合法的 volume 名: %s", source)
		}
		return mount.Mount{
			Type:     mount.TypeVolume,
			Source:   source,
			Target:   vol.ContainerPath,
			ReadOnly: vol.ReadOnly,
		}, nil
	case string(mount.TypeBind):
		bindSource, err := s.resolveBindSource(containerName, source)
		if err != nil {
			return mount.Mount{}, err
		}
		return mount.Mount{
			Type:        mount.TypeBind,
			Source:      bindSource,
			Target:      vol.ContainerPath,
			ReadOnly:    vol.ReadOnly,
			BindOptions: &mount.BindOptions{CreateMountpoint: true},
		}, nil
	default:
		return mount.Mount{}, fmt.Errorf("不支持的挂载类型: %s", mountType)
	}
}

func (s *DockerService) resolveBindSource(containerName string, source string) (string, error) {
	bindSource := source
	if s.config.ContainerRoot != "" && !filepath.IsAbs(bindSource) {
		bindSource = filepath.Join(s.config.ContainerRoot, containerName, bindSource)
	}

	if _, err := os.Stat(bindSource); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("检查挂载源失败: %w", err)
	}
	return bindSource, nil
}

func inferMountType(source string) string {
	if filepath.IsAbs(source) || strings.HasPrefix(source, ".") {
		return string(mount.TypeBind)
	}
	return string(mount.TypeVolume)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
