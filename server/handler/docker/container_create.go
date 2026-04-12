package docker

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"
	"isrvd/server/model"
)

// CreateContainer 创建容器
func (h *DockerHandler) CreateContainer(c *gin.Context) {
	var req model.ContainerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := c.Request.Context()

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
		hostConfig.Memory = req.Memory * 1024 * 1024 // MB to Bytes
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
			// 如果配置了容器数据根目录且 hostPath 是相对路径，则补全为容器专属目录
			if config.ContainerRoot != "" && !filepath.IsAbs(hostPath) {
				hostPath = filepath.Join(config.ContainerRoot, req.Name, hostPath)
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

	resp, err := h.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, req.Name)
	if err != nil {
		logman.Error("Create container failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建容器失败: "+err.Error())
		return
	}

	h.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

	// 生成 docker-compose.yml 配置文件
	if req.Name != "" && config.ContainerRoot != "" {
		if err := h.createComposeFile(req); err != nil {
			logman.Warn("Failed to create compose file", "error", err)
		}
	}

	logman.Info("Container created", "id", resp.ID[:12], "name", req.Name)
	helper.RespondSuccess(c, "容器创建成功", gin.H{"id": resp.ID[:12], "name": req.Name})
}

// UpdateContainerConfig 更新容器配置并重建
func (h *DockerHandler) UpdateContainerConfig(c *gin.Context) {
	var req model.ContainerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Name == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器名称不能为空")
		return
	}

	ctx := c.Request.Context()

	// 查找并停止旧容器
	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败: "+err.Error())
		return
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
		_ = h.dockerClient.ContainerStop(ctx, oldContainerID, container.StopOptions{Timeout: &timeout})
		_ = h.dockerClient.ContainerRemove(ctx, oldContainerID, types.ContainerRemoveOptions{Force: true})
	}

	// 创建新容器配置
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
			if config.ContainerRoot != "" && !filepath.IsAbs(hostPath) {
				hostPath = filepath.Join(config.ContainerRoot, req.Name, hostPath)
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

	// 创建新容器
	resp, err := h.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, req.Name)
	if err != nil {
		logman.Error("Recreate container failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "重建容器失败: "+err.Error())
		return
	}

	// 启动新容器
	h.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

	// 更新 compose 配置文件
	if config.ContainerRoot != "" {
		createReq := model.ContainerCreateRequest{
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
		if err := h.createComposeFile(createReq); err != nil {
			logman.Warn("Failed to update compose file", "error", err)
		}
	}

	logman.Info("Container recreated", "id", resp.ID[:12], "name", req.Name)
	helper.RespondSuccess(c, "容器配置更新成功，已重建容器", gin.H{"id": resp.ID[:12], "name": req.Name})
}

// GetContainerConfig 获取容器配置（从 compose 文件读取）
func (h *DockerHandler) GetContainerConfig(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器名称不能为空")
		return
	}

	compose, err := helper.ReadComposeFile(config.ContainerRoot, name)
	if err != nil {
		// compose 文件不存在，尝试根据容器当前配置自动创建
		compose, err = h.autoCreateComposeFile(c.Request.Context(), name)
		if err != nil {
			helper.RespondError(c, http.StatusNotFound, "未找到容器配置文件且无法自动生成: "+err.Error())
			return
		}
		logman.Info("Auto-created compose file for container", "name", name)
	}

	service, ok := compose.Services[name]
	if !ok {
		helper.RespondError(c, http.StatusNotFound, "配置文件中未找到该容器")
		return
	}

	// 转换为前端需要的格式
	volumes := helper.ParseVolumesToList(service.Volumes)
	modelVolumes := make([]model.VolumeMapping, len(volumes))
	for i, v := range volumes {
		modelVolumes[i] = model.VolumeMapping{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		}
	}

	result := model.ContainerConfigResponse{
		Image:      service.Image,
		Name:       name,
		Env:        service.Environment,
		Ports:      helper.ParsePortsToMap(service.Ports),
		Volumes:    modelVolumes,
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
			// 解析内存限制，如 "512M" -> 512
			memStr := strings.TrimSuffix(service.Deploy.Resources.Limits.Memory, "M")
			memStr = strings.TrimSuffix(memStr, "Mi")
			if mem, err := parseInt(memStr); err == nil {
				result.Memory = mem
			}
		}
		if service.Deploy.Resources.Limits.Cpus != "" {
			if cpus, err := parseFloat(service.Deploy.Resources.Limits.Cpus); err == nil {
				result.Cpus = cpus
			}
		}
	}

	helper.RespondSuccess(c, "获取容器配置成功", result)
}
