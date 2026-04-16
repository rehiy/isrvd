package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rehiy/pango/logman"
)

// VolumeMapping 目录映射
type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

// composeService 定义 docker-compose service 配置（包内私有）
type composeService struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Environment   []string          `yaml:"environment,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	NetworkMode   string            `yaml:"network_mode,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
	Command       string            `yaml:"command,omitempty"`
	Entrypoint    string            `yaml:"entrypoint,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	User          string            `yaml:"user,omitempty"`
	Hostname      string            `yaml:"hostname,omitempty"`
	Privileged    bool              `yaml:"privileged,omitempty"`
	CapAdd        []string          `yaml:"cap_add,omitempty"`
	CapDrop       []string          `yaml:"cap_drop,omitempty"`
	Deploy        *composeDeploy    `yaml:"deploy,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
}

// composeDeploy 定义资源限制配置
type composeDeploy struct {
	Resources *composeResources `yaml:"resources,omitempty"`
}

// composeResources 定义资源配置
type composeResources struct {
	Limits *composeLimit `yaml:"limits,omitempty"`
}

// composeLimit 定义资源限制
type composeLimit struct {
	Cpus   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// composeFile 定义 docker-compose 文件结构
type composeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]composeService `yaml:"services"`
}

// AutoCreateComposeFile 根据容器当前运行配置自动生成 compose 文件
func (s *DockerService) AutoCreateComposeFile(ctx context.Context, name string) (*composeFile, error) {
	// 通过容器名查找容器
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	var containerID string
	for _, ct := range containers {
		ctName := ""
		if len(ct.Names) > 0 {
			ctName = strings.TrimPrefix(ct.Names[0], "/")
		}
		if ctName == name {
			containerID = ct.ID
			break
		}
	}

	if containerID == "" {
		return nil, fmt.Errorf("未找到名称为 %s 的容器", name)
	}

	// 检查容器详细配置
	inspect, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("检查容器配置失败: %w", err)
	}

	// 拒绝为 Swarm 管理的容器生成 compose 文件
	if inspect.Config.Labels["com.docker.swarm.service.id"] != "" {
		return nil, fmt.Errorf("容器 %s 由 Docker Swarm 管理，不支持生成 compose 文件", name)
	}

	// 过滤掉镜像内置的默认环境变量，只保留用户自定义的
	var filteredEnv []string
	imageInspect, _, imgErr := s.client.ImageInspectWithRaw(ctx, inspect.Config.Image)
	if imgErr == nil {
		defaultEnvSet := make(map[string]bool, len(imageInspect.Config.Env))
		for _, env := range imageInspect.Config.Env {
			defaultEnvSet[env] = true
		}
		for _, env := range inspect.Config.Env {
			if strings.HasPrefix(env, "HOSTNAME=") {
				continue
			}
			if !defaultEnvSet[env] {
				filteredEnv = append(filteredEnv, env)
			}
		}
	} else {
		for _, env := range inspect.Config.Env {
			if strings.HasPrefix(env, "HOSTNAME=") {
				continue
			}
			filteredEnv = append(filteredEnv, env)
		}
	}

	// 过滤 Hostname：默认值为容器ID，仅在用户明确设置时保留
	hostname := inspect.Config.Hostname
	if hostname == containerID || (len(containerID) >= 12 && hostname == containerID[:12]) {
		hostname = ""
	}

	service := composeService{
		Image:         inspect.Config.Image,
		ContainerName: name,
		Environment:   filteredEnv,
		WorkingDir:    inspect.Config.WorkingDir,
		User:          inspect.Config.User,
		Hostname:      hostname,
		Labels:        make(map[string]string),
	}

	// 处理命令
	if len(inspect.Config.Cmd) > 0 {
		service.Command = strings.Join(inspect.Config.Cmd, " ")
	}

	// 处理 Entrypoint：仅在与镜像默认值不同时保留
	if len(inspect.Config.Entrypoint) > 0 {
		entrypoint := strings.Join(inspect.Config.Entrypoint, " ")
		if imgErr == nil && len(imageInspect.Config.Entrypoint) > 0 {
			defaultEntrypoint := strings.Join(imageInspect.Config.Entrypoint, " ")
			if entrypoint != defaultEntrypoint {
				service.Entrypoint = entrypoint
			}
		} else {
			service.Entrypoint = entrypoint
		}
	}

	// 处理 Labels：过滤掉 Docker 自动添加的内部标签
	for k, v := range inspect.Config.Labels {
		if strings.HasPrefix(k, "com.docker.") || strings.HasPrefix(k, "org.opencontainers.") {
			continue
		}
		service.Labels[k] = v
	}
	if len(service.Labels) == 0 {
		service.Labels = nil
	}

	// 处理重启策略
	restartPolicy := inspect.HostConfig.RestartPolicy.Name
	switch restartPolicy {
	case "always", "on-failure", "unless-stopped":
		service.Restart = restartPolicy
	default:
		service.Restart = "no"
	}

	// 处理网络模式
	networkMode := string(inspect.HostConfig.NetworkMode)
	if networkMode != "" && networkMode != "default" {
		service.NetworkMode = networkMode
	}

	// 处理端口映射
	if len(inspect.HostConfig.PortBindings) > 0 {
		service.Ports = make([]string, 0)
		for port, bindings := range inspect.HostConfig.PortBindings {
			portNum := strings.TrimSuffix(string(port), "/tcp")
			portNum = strings.TrimSuffix(portNum, "/udp")
			for _, binding := range bindings {
				if binding.HostPort != "" {
					service.Ports = append(service.Ports, fmt.Sprintf("%s:%s", binding.HostPort, portNum))
				}
			}
		}
	}

	// 处理卷映射：统一从 Mounts 提取，支持 bind mount 和 named volume
	if len(inspect.Mounts) > 0 {
		service.Volumes = make([]string, 0, len(inspect.Mounts))
		for _, mount := range inspect.Mounts {
			if mount.Type == "bind" {
				if mount.Source == "" {
					continue
				}
				bind := mount.Source + ":" + mount.Destination
				if !mount.RW {
					bind += ":ro"
				}
				service.Volumes = append(service.Volumes, bind)
			} else if mount.Type == "volume" {
				if mount.Name == "" {
					continue
				}
				vol := mount.Name + ":" + mount.Destination
				if !mount.RW {
					vol += ":ro"
				}
				service.Volumes = append(service.Volumes, vol)
			}
		}
	}

	// 处理资源限制
	if inspect.HostConfig.Memory > 0 || inspect.HostConfig.NanoCPUs > 0 {
		service.Deploy = &composeDeploy{
			Resources: &composeResources{
				Limits: &composeLimit{},
			},
		}
		if inspect.HostConfig.Memory > 0 {
			memoryMB := inspect.HostConfig.Memory / (1024 * 1024)
			if memoryMB > 0 {
				service.Deploy.Resources.Limits.Memory = fmt.Sprintf("%dM", memoryMB)
			}
		}
		if inspect.HostConfig.NanoCPUs > 0 {
			cpus := float64(inspect.HostConfig.NanoCPUs) / 1e9
			service.Deploy.Resources.Limits.Cpus = fmt.Sprintf("%.1f", cpus)
		}
	}

	// 处理安全配置
	if inspect.HostConfig.Privileged {
		service.Privileged = true
	}
	if len(inspect.HostConfig.CapAdd) > 0 {
		service.CapAdd = inspect.HostConfig.CapAdd
	}
	if len(inspect.HostConfig.CapDrop) > 0 {
		service.CapDrop = inspect.HostConfig.CapDrop
	}

	// 创建 compose 文件
	if err := createComposeFileOnDisk(s.config.ContainerRoot, name, service); err != nil {
		return nil, fmt.Errorf("自动创建 compose 文件失败: %w", err)
	}

	return readComposeFileFromDisk(s.config.ContainerRoot, name)
}

// CreateComposeFile 根据 ContainerCreateRequest 生成 compose 配置文件
func (s *DockerService) CreateComposeFile(req ContainerCreateRequest) error {
	service := composeService{
		Image:         req.Image,
		ContainerName: req.Name,
		Environment:   req.Env,
		NetworkMode:   req.Network,
		WorkingDir:    req.Workdir,
		User:          req.User,
		Hostname:      req.Hostname,
		Labels:        make(map[string]string),
	}

	if len(req.Cmd) > 0 {
		service.Command = strings.Join(req.Cmd, " ")
	}

	if req.Restart != "" {
		service.Restart = req.Restart
	} else {
		service.Restart = "always"
	}

	// 处理端口映射
	if len(req.Ports) > 0 {
		service.Ports = make([]string, 0, len(req.Ports))
		for hostPort, containerPort := range req.Ports {
			service.Ports = append(service.Ports, fmt.Sprintf("%s:%s", hostPort, containerPort))
		}
	}

	// 处理卷映射
	if len(req.Volumes) > 0 {
		service.Volumes = make([]string, 0, len(req.Volumes))
		for _, vol := range req.Volumes {
			hostPath := vol.HostPath
			if s.config.ContainerRoot != "" && !filepath.IsAbs(hostPath) {
				hostPath = filepath.Join(s.config.ContainerRoot, req.Name, hostPath)
			}
			bind := hostPath + ":" + vol.ContainerPath
			if vol.ReadOnly {
				bind += ":ro"
			}
			service.Volumes = append(service.Volumes, bind)
		}
	}

	// 处理资源限制
	if req.Memory > 0 || req.Cpus > 0 {
		service.Deploy = &composeDeploy{
			Resources: &composeResources{
				Limits: &composeLimit{},
			},
		}
		if req.Memory > 0 {
			service.Deploy.Resources.Limits.Memory = fmt.Sprintf("%dM", req.Memory)
		}
		if req.Cpus > 0 {
			service.Deploy.Resources.Limits.Cpus = fmt.Sprintf("%.1f", req.Cpus)
		}
	}

	// 处理安全配置
	if req.Privileged {
		service.Privileged = true
	}
	if len(req.CapAdd) > 0 {
		service.CapAdd = req.CapAdd
	}
	if len(req.CapDrop) > 0 {
		service.CapDrop = req.CapDrop
	}

	return createComposeFileOnDisk(s.config.ContainerRoot, req.Name, service)
}

// ContainerConfigResponse 容器配置响应（从 compose 文件读取）
type ContainerConfigResponse struct {
	Image      string            `json:"image"`
	Name       string            `json:"name"`
	Cmd        []string          `json:"cmd,omitempty"`
	Env        []string          `json:"env,omitempty"`
	Ports      map[string]string `json:"ports,omitempty"`
	Volumes    []VolumeMapping   `json:"volumes,omitempty"`
	Network    string            `json:"network,omitempty"`
	Restart    string            `json:"restart,omitempty"`
	Memory     int64             `json:"memory,omitempty"`
	Cpus       float64           `json:"cpus,omitempty"`
	Workdir    string            `json:"workdir,omitempty"`
	User       string            `json:"user,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Privileged bool              `json:"privileged,omitempty"`
	CapAdd     []string          `json:"capAdd,omitempty"`
	CapDrop    []string          `json:"capDrop,omitempty"`
}

// GetContainerConfig 获取容器配置（从 compose 文件读取）
func (s *DockerService) GetContainerConfig(ctx context.Context, name string) (*ContainerConfigResponse, error) {
	compose, err := readComposeFileFromDisk(s.config.ContainerRoot, name)
	if err != nil {
		// compose 文件不存在，尝试根据容器当前配置自动创建
		compose, err = s.AutoCreateComposeFile(ctx, name)
		if err != nil {
			logman.Error("Get container config failed", "name", name, "error", err)
			return nil, fmt.Errorf("容器配置未找到且无法自动生成: %w", err)
		}
	}

	service, ok := compose.Services[name]
	if !ok {
		return nil, fmt.Errorf("配置文件中未找到该容器")
	}

	// 转换为前端需要的格式
	volumes := parseVolumesToList(service.Volumes)

	result := &ContainerConfigResponse{
		Image:      service.Image,
		Name:       name,
		Env:        service.Environment,
		Ports:      parsePortsToMap(service.Ports),
		Volumes:    volumes,
		Network:    service.NetworkMode,
		Restart:    service.Restart,
		Workdir:    service.WorkingDir,
		User:       service.User,
		Hostname:   service.Hostname,
		Privileged: service.Privileged,
		CapAdd:     service.CapAdd,
		CapDrop:    service.CapDrop,
	}

	if service.Command != "" {
		result.Cmd = strings.Fields(service.Command)
	}

	if service.Deploy != nil && service.Deploy.Resources != nil && service.Deploy.Resources.Limits != nil {
		if service.Deploy.Resources.Limits.Memory != "" {
			memStr := strings.TrimSuffix(service.Deploy.Resources.Limits.Memory, "M")
			memStr = strings.TrimSuffix(memStr, "Mi")
			if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
				result.Memory = mem
			}
		}
		if service.Deploy.Resources.Limits.Cpus != "" {
			if cpus, err := strconv.ParseFloat(service.Deploy.Resources.Limits.Cpus, 64); err == nil {
				result.Cpus = cpus
			}
		}
	}

	return result, nil
}
