package docker

import (
	"context"
	"fmt"
	"strings"

	pkgdocker "isrvd/pkgs/docker"
)

// ContainerCreateResult 创建容器结果
type ContainerCreateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContainerUpdateResult 更新容器结果
type ContainerUpdateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContainerLogsResult 容器日志结果
type ContainerLogsResult struct {
	ID   string   `json:"id"`
	Logs []string `json:"logs"`
}

// ListContainers 列出容器
func (s *Service) ListContainers(ctx context.Context, all bool) (any, error) {
	return s.docker.ListContainers(ctx, all)
}

// CreateContainer 创建容器
func (s *Service) CreateContainer(ctx context.Context, req pkgdocker.ContainerCreateRequest) (*ContainerCreateResult, error) {
	id, err := s.docker.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	s.saveSnapshot(req)
	shortID := id
	if len(id) > 12 {
		shortID = id[:12]
	}
	return &ContainerCreateResult{ID: shortID, Name: req.Name}, nil
}

// UpdateContainerConfig 更新容器配置并重建
func (s *Service) UpdateContainerConfig(ctx context.Context, req pkgdocker.ContainerUpdateRequest) (*ContainerUpdateResult, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("容器名称不能为空")
	}
	id, err := s.docker.UpdateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	s.saveSnapshot(req.ToCreateRequest())
	shortID := id
	if len(id) > 12 {
		shortID = id[:12]
	}
	return &ContainerUpdateResult{ID: shortID, Name: req.Name}, nil
}

// GetContainerConfig 获取容器配置（从 Docker inspect 读取，排除镜像内置默认值）
func (s *Service) GetContainerConfig(ctx context.Context, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("容器名称不能为空")
	}
	info, err := s.docker.InspectContainer(ctx, name)
	if err != nil {
		return nil, err
	}

	// 获取镜像内置默认配置，用于差集计算
	var imgCmd []string
	var imgEnvSet map[string]struct{}
	var imgWorkdir string
	if imgInfo, imgErr := s.docker.InspectImage(ctx, info.Config.Image); imgErr == nil {
		imgCmd = imgInfo.Cmd
		imgWorkdir = imgInfo.WorkingDir
		// 构建镜像内置 env 集合（key=value 完整匹配）
		imgEnvSet = make(map[string]struct{}, len(imgInfo.Env))
		for _, e := range imgInfo.Env {
			imgEnvSet[e] = struct{}{}
		}
	} else {
		imgEnvSet = map[string]struct{}{}
	}

	// 解析端口映射
	ports := map[string]string{}
	if info.HostConfig != nil {
		for containerPort, bindings := range info.HostConfig.PortBindings {
			if len(bindings) > 0 {
				cp := strings.TrimSuffix(string(containerPort), "/tcp")
				cp = strings.TrimSuffix(cp, "/udp")
				ports[bindings[0].HostPort] = cp
			}
		}
	}

	// 解析目录映射
	var volumes []pkgdocker.VolumeMapping
	if info.HostConfig != nil {
		for _, bind := range info.HostConfig.Binds {
			parts := strings.Split(bind, ":")
			if len(parts) >= 2 {
				vm := pkgdocker.VolumeMapping{
					HostPath:      parts[0],
					ContainerPath: parts[1],
					ReadOnly:      len(parts) >= 3 && parts[2] == "ro",
				}
				volumes = append(volumes, vm)
			}
		}
	}

	// 解析重启策略
	restart := "no"
	if info.HostConfig != nil {
		restart = string(info.HostConfig.RestartPolicy.Name)
	}

	// 解析资源限制
	var memory int64
	var cpus float64
	if info.HostConfig != nil {
		if info.HostConfig.Memory > 0 {
			memory = info.HostConfig.Memory / 1024 / 1024
		}
		if info.HostConfig.NanoCPUs > 0 {
			cpus = float64(info.HostConfig.NanoCPUs) / 1e9
		}
	}

	// 解析安全配置
	privileged := false
	var capAdd, capDrop []string
	if info.HostConfig != nil {
		privileged = info.HostConfig.Privileged
		capAdd = info.HostConfig.CapAdd
		capDrop = info.HostConfig.CapDrop
	}

	// 排除镜像内置的 Env（只保留用户额外添加的）
	var userEnv []string
	for _, e := range info.Config.Env {
		if _, ok := imgEnvSet[e]; !ok {
			userEnv = append(userEnv, e)
		}
	}

	// 排除镜像内置的 Cmd（若与镜像默认相同则置空，表示未覆盖）
	var userCmd []string
	if !stringSliceEqual(info.Config.Cmd, imgCmd) {
		userCmd = info.Config.Cmd
	}

	// 排除镜像内置的 WorkingDir（若与镜像默认相同则置空）
	workdir := info.Config.WorkingDir
	if workdir == imgWorkdir {
		workdir = ""
	}

	// Hostname：若与容器 ID 前12位相同（Docker 自动生成）则置空
	hostname := info.Config.Hostname
	shortID := info.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	if hostname == shortID {
		hostname = ""
	}

	result := &pkgdocker.ContainerCreateRequest{
		Name:       strings.TrimPrefix(info.Name, "/"),
		Image:      info.Config.Image,
		Cmd:        userCmd,
		Env:        userEnv,
		Ports:      ports,
		Volumes:    volumes,
		Restart:    restart,
		Network:    string(info.HostConfig.NetworkMode),
		Memory:     memory,
		Cpus:       cpus,
		Workdir:    workdir,
		User:       info.Config.User,
		Hostname:   hostname,
		Privileged: privileged,
		CapAdd:     capAdd,
		CapDrop:    capDrop,
	}
	return result, nil
}

// GetContainerCompose 获取容器对应的 compose 文件内容（YAML 文本）
// 快照缺失时自动从运行态反推生成
func (s *Service) GetContainerCompose(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("容器名称不能为空")
	}
	if s.composeReader == nil {
		return "", fmt.Errorf("compose 服务不可用")
	}
	return s.composeReader.GetComposeContent(ctx, name)
}

// stringSliceEqual 比较两个字符串切片是否相等
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	return s.docker.GetContainerStats(ctx, id)
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req pkgdocker.ContainerActionRequest) error {
	return s.docker.ContainerAction(ctx, req.ID, req.Action)
}

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req pkgdocker.ContainerLogsRequest) (*ContainerLogsResult, error) {
	logs, err := s.docker.GetContainerLogs(ctx, req.ID, req.Tail)
	if err != nil {
		return nil, err
	}
	return &ContainerLogsResult{ID: req.ID, Logs: logs}, nil
}

// Info 获取 Docker 概览信息
func (s *Service) Info(ctx context.Context) (any, error) {
	return s.docker.GetInfo(ctx)
}
