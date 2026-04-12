package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"

	"isrvd/server/config"
	"isrvd/server/helper"
	"isrvd/server/model"
)

// autoCreateComposeFile 根据容器当前运行配置自动生成 compose 文件
func (h *DockerHandler) autoCreateComposeFile(ctx context.Context, name string) (*helper.ComposeFile, error) {
	// 通过容器名查找容器
	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
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
	inspect, err := h.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("检查容器配置失败: %w", err)
	}

	// 过滤掉镜像内置的默认环境变量，只保留用户自定义的
	var filteredEnv []string
	imageInspect, _, imgErr := h.dockerClient.ImageInspectWithRaw(ctx, inspect.Config.Image)
	if imgErr == nil {
		// 构建镜像默认环境变量的集合（完整 KEY=VALUE 匹配）
		defaultEnvSet := make(map[string]bool, len(imageInspect.Config.Env))
		for _, env := range imageInspect.Config.Env {
			defaultEnvSet[env] = true
		}
		// 过滤：只保留不在镜像默认环境变量中的条目
		for _, env := range inspect.Config.Env {
			if strings.HasPrefix(env, "HOSTNAME=") {
				continue
			}
			if !defaultEnvSet[env] {
				filteredEnv = append(filteredEnv, env)
			}
		}
	} else {
		// 如果无法检查镜像，则仅排除 HOSTNAME
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

	service := helper.ComposeService{
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
		// 跳过 Docker 自动添加的标签
		if strings.HasPrefix(k, "com.docker.") || strings.HasPrefix(k, "org.opencontainers.") {
			continue
		}
		service.Labels[k] = v
	}
	// 如果过滤后没有标签，置为 nil 避免输出空 map
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
				// 命名卷格式：volume_name:container_path
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
		service.Deploy = &helper.ComposeDeploy{
			Resources: &helper.ComposeResources{
				Limits: &helper.ComposeLimit{},
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
	if err := helper.CreateComposeFile(config.ContainerRoot, name, service); err != nil {
		return nil, fmt.Errorf("自动创建 compose 文件失败: %w", err)
	}

	return helper.ReadComposeFile(config.ContainerRoot, name)
}

// createComposeFile 根据 ContainerCreateRequest 生成 compose 配置文件
func (h *DockerHandler) createComposeFile(req model.ContainerCreateRequest) error {
	service := helper.ComposeService{
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
			// 如果配置了容器数据根目录且 hostPath 是相对路径，则补全为容器专属目录
			if config.ContainerRoot != "" && !filepath.IsAbs(hostPath) {
				hostPath = filepath.Join(config.ContainerRoot, req.Name, hostPath)
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
		service.Deploy = &helper.ComposeDeploy{
			Resources: &helper.ComposeResources{
				Limits: &helper.ComposeLimit{},
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

	return helper.CreateComposeFile(config.ContainerRoot, req.Name, service)
}
