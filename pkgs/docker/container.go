package docker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/rehiy/pango/logman"
)

// ListContainers 获取容器列表
func (s *DockerService) ListContainers(ctx context.Context, all bool) ([]*ContainerInfo, error) {
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{All: all})
	if err != nil {
		logman.Error("List containers failed", "error", err)
		return nil, err
	}

	var result []*ContainerInfo
	for _, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, &ContainerInfo{
			ID:      ct.ID[:12],
			Name:    name,
			Image:   ct.Image,
			State:   ct.State,
			Status:  ct.Status,
			Ports:   formatPorts(ct.Ports),
			Created: ct.Created,
			IsSwarm: ct.Labels["com.docker.swarm.service.id"] != "",
			Labels:  ct.Labels,
		})
	}

	return result, nil
}

// ContainerAction 容器操作（start/stop/restart/remove/pause/unpause）
func (s *DockerService) ContainerAction(ctx context.Context, id, action string) error {
	var err error
	switch action {
	case "start":
		err = s.client.ContainerStart(ctx, id, types.ContainerStartOptions{})
	case "stop":
		timeout := 10
		err = s.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
	case "restart":
		timeout := 10
		err = s.client.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
	case "remove":
		err = s.client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true})
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

// GetContainerLogs 获取容器日志
func (s *DockerService) GetContainerLogs(ctx context.Context, id, tail string) ([]string, error) {
	if tail == "" {
		tail = "100"
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true,
		Tail: tail, Follow: false, Timestamps: true,
	}

	reader, err := s.client.ContainerLogs(ctx, id, options)
	if err != nil {
		logman.Error("Get container logs failed", "id", id, "error", err)
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logman.Error("Read container logs failed", "id", id, "error", err)
		return nil, err
	}

	return ParseDockerLogs(data), nil
}

// CreateContainer 创建容器
func (s *DockerService) CreateContainer(ctx context.Context, req ContainerCreateRequest) (string, error) {
	containerConfig := &container.Config{
		Image:      req.Image,
		Cmd:        req.Cmd,
		Env:        req.Env,
		WorkingDir: req.Workdir,
		User:       req.User,
		Hostname:   req.Hostname,
	}

	hostConfig := &container.HostConfig{}

	// 处理重启策略
	switch req.Restart {
	case "always":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "always"}
	case "on-failure":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "on-failure"}
	case "unless-stopped":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
	default:
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "no"}
	}

	// 处理网络模式
	if req.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(req.Network)
	}

	// 处理资源限制
	if req.Memory > 0 {
		hostConfig.Memory = req.Memory * 1024 * 1024
	}
	if req.Cpus > 0 {
		hostConfig.NanoCPUs = int64(req.Cpus * 1e9)
	}

	// 处理端口映射
	if len(req.Ports) > 0 {
		portBindings := make(nat.PortMap)
		for hostPort, containerPort := range req.Ports {
			port := nat.Port(containerPort + "/tcp")
			portBindings[port] = []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			}
		}
		hostConfig.PortBindings = portBindings
		containerConfig.ExposedPorts = make(nat.PortSet)
		for _, containerPort := range req.Ports {
			containerConfig.ExposedPorts[nat.Port(containerPort+"/tcp")] = struct{}{}
		}
	}

	// 处理目录映射
	if len(req.Volumes) > 0 {
		hostConfig.Binds = make([]string, 0, len(req.Volumes))
		for _, vol := range req.Volumes {
			hostPath := vol.HostPath
			if s.config.ContainerRoot != "" && !filepath.IsAbs(hostPath) {
				hostPath = filepath.Join(s.config.ContainerRoot, req.Name, hostPath)
			}
			bind := hostPath + ":" + vol.ContainerPath
			if vol.ReadOnly {
				bind += ":ro"
			}
			hostConfig.Binds = append(hostConfig.Binds, bind)
		}
	}

	// 处理安全配置
	if req.Privileged {
		hostConfig.Privileged = true
	}
	if len(req.CapAdd) > 0 {
		hostConfig.CapAdd = req.CapAdd
	}
	if len(req.CapDrop) > 0 {
		hostConfig.CapDrop = req.CapDrop
	}

	resp, err := s.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, req.Name)
	if err != nil {
		logman.Error("Create container failed", "name", req.Name, "error", err)
		return "", err
	}

	// 启动容器
	s.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

	// 生成 docker-compose.yml 配置文件
	if req.Name != "" && s.config.ContainerRoot != "" {
		if err := s.CreateComposeFile(req); err != nil {
			logman.Warn("Failed to create compose file", "error", err)
		}
	}

	shortID := resp.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	logman.Info("Container created", "id", shortID, "name", req.Name)

	return resp.ID, nil
}

// UpdateContainer 更新容器配置并重建
func (s *DockerService) UpdateContainer(ctx context.Context, req ContainerUpdateRequest) (string, error) {
	// 查找并停止旧容器
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("获取容器列表失败: %w", err)
	}

	var oldContainerID string
	for _, ct := range containers {
		ctName := ""
		if len(ct.Names) > 0 {
			ctName = strings.TrimPrefix(ct.Names[0], "/")
		}
		if ctName == req.Name {
			oldContainerID = ct.ID
			break
		}
	}

	// 停止并删除旧容器
	if oldContainerID != "" {
		timeout := 10
		_ = s.client.ContainerStop(ctx, oldContainerID, container.StopOptions{Timeout: &timeout})
		_ = s.client.ContainerRemove(ctx, oldContainerID, types.ContainerRemoveOptions{Force: true})
	}

	// 转换为 CreateRequest 复用创建逻辑
	createReq := ContainerCreateRequest{
		Image:      req.Image,
		Name:       req.Name,
		Cmd:        req.Cmd,
		Env:        req.Env,
		Ports:      req.Ports,
		Volumes:    req.Volumes,
		Network:    req.Network,
		Restart:    req.Restart,
		Memory:     req.Memory,
		Cpus:       req.Cpus,
		Workdir:    req.Workdir,
		User:       req.User,
		Hostname:   req.Hostname,
		Privileged: req.Privileged,
		CapAdd:     req.CapAdd,
		CapDrop:    req.CapDrop,
	}

	return s.CreateContainer(ctx, createReq)
}
