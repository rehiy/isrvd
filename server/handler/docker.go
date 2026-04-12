package handler

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rehiy/pango/logman"
	"github.com/shirou/gopsutil/v3/cpu"

	"isrvd/server/config"
	"isrvd/server/helper"
	"isrvd/server/model"
)

// Docker处理器
type DockerHandler struct {
	dockerClient *client.Client
}

// 创建Docker处理器
func NewDockerHandler() (*DockerHandler, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerHandler{dockerClient: cli}, nil
}

func (h *DockerHandler) GetClient() *client.Client {
	return h.dockerClient
}

// ==================== 概览 ====================

func (h *DockerHandler) Info(c *gin.Context) {
	ctx := context.Background()

	info, err := h.dockerClient.Info(ctx)
	if err != nil {
		logman.Error("Docker info failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法连接到 Docker 服务: "+err.Error())
		return
	}
	_ = info

	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		logman.Error("Container list failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败")
		return
	}

	var running, stopped, paused int64
	for _, ct := range containers {
		switch ct.State {
		case "running":
			running++
		case "paused":
			paused++
		default:
			stopped++
		}
	}

	images, _ := h.dockerClient.ImageList(ctx, types.ImageListOptions{All: true})
	volList, _ := h.dockerClient.VolumeList(ctx, volume.ListOptions{})
	networks, _ := h.dockerClient.NetworkList(ctx, types.NetworkListOptions{})

	helper.RespondSuccess(c, "Docker info retrieved", model.DockerInfo{
		ContainersRunning: running,
		ContainersStopped: stopped,
		ContainersPaused:  paused,
		ImagesTotal:       int64(len(images)),
		VolumesTotal:      int64(len(volList.Volumes)),
		NetworksTotal:     int64(len(networks)),
	})
}

// ==================== 容器管理 ====================

func (h *DockerHandler) ListContainers(c *gin.Context) {
	ctx := context.Background()
	all := c.DefaultQuery("all", "false") == "true"

	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: all})
	if err != nil {
		logman.Error("List containers failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败")
		return
	}

	var result []*model.ContainerInfo
	for _, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, &model.ContainerInfo{
			ID:      ct.ID[:12],
			Name:    name,
			Image:   ct.Image,
			State:   ct.State,
			Status:  ct.Status,
			Ports:   formatPorts(ct.Ports),
			Created: ct.Created,
			Labels:  ct.Labels,
		})
	}

	helper.RespondSuccess(c, "Containers listed successfully", result)
}

func (h *DockerHandler) ContainerAction(c *gin.Context) {
	var req model.ContainerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()

	switch req.Action {
	case "start":
		err := h.dockerClient.ContainerStart(ctx, req.ID, types.ContainerStartOptions{})
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "启动容器失败: "+err.Error())
			return
		}
	case "stop":
		timeout := 10
		err := h.dockerClient.ContainerStop(ctx, req.ID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "停止容器失败: "+err.Error())
			return
		}
	case "restart":
		timeout := 10
		err := h.dockerClient.ContainerRestart(ctx, req.ID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "重启容器失败: "+err.Error())
			return
		}
	case "remove":
		err := h.dockerClient.ContainerRemove(ctx, req.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "删除容器失败: "+err.Error())
			return
		}
	case "pause":
		err := h.dockerClient.ContainerPause(ctx, req.ID)
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "暂停容器失败: "+err.Error())
			return
		}
	case "unpause":
		err := h.dockerClient.ContainerUnpause(ctx, req.ID)
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "恢复容器失败: "+err.Error())
			return
		}
	default:
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}

	actionName := map[string]string{
		"start": "启动", "stop": "停止", "restart": "重启",
		"remove": "删除", "pause": "暂停", "unpause": "恢复",
	}[req.Action]

	logman.Info("Container action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, actionName+"操作成功", nil)
}

func (h *DockerHandler) CreateContainer(c *gin.Context) {
	var req model.ContainerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()

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

func (h *DockerHandler) ContainerLogs(c *gin.Context) {
	var req model.ContainerLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()

	tailStr := req.Tail
	if tailStr == "" {
		tailStr = "100"
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true,
		Tail: tailStr, Follow: false, Timestamps: true,
	}

	reader, err := h.dockerClient.ContainerLogs(ctx, req.ID, options)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取日志失败: "+err.Error())
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "读取日志失败")
		return
	}

	helper.RespondSuccess(c, "Container logs retrieved", gin.H{"id": req.ID, "logs": parseDockerLogs(data)})
}

// ==================== 镜像管理 ====================

func (h *DockerHandler) ListImages(c *gin.Context) {
	ctx := context.Background()
	all := c.DefaultQuery("all", "false") == "true"

	images, err := h.dockerClient.ImageList(ctx, types.ImageListOptions{All: all})
	if err != nil {
		logman.Error("List images failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取镜像列表失败")
		return
	}

	var result []*model.ImageInfo
	for _, img := range images {
		// 过滤掉中间层镜像（没有 RepoTags 的镜像）
		if !all && len(img.RepoTags) == 0 {
			continue
		}

		id := img.ID
		// 计算短ID用于显示
		shortID := id
		if len(id) > 7 && strings.HasPrefix(id, "sha256:") {
			shortID = id[7:min(19, len(id))]
		} else if len(id) > 12 {
			shortID = id[:12]
		}
		result = append(result, &model.ImageInfo{
			ID: id, ShortID: shortID, RepoTags: img.RepoTags,
			Size: img.Size, Created: img.Created,
		})
	}

	helper.RespondSuccess(c, "Images listed successfully", result)
}

func (h *DockerHandler) ImageAction(c *gin.Context) {
	var req model.ImageActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	switch req.Action {
	case "remove":
		// 直接使用传入的ID（已经是完整的sha256:xxx格式）
		// Force: 强制删除，即使镜像有多个标签引用
		// PruneChildren: 自动删除未被其他镜像引用的子镜像（中间层镜像）
		_, err := h.dockerClient.ImageRemove(ctx, req.ID, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "删除镜像失败: "+err.Error())
			return
		}
	default:
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Image action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, "镜像操作成功", nil)
}

func (h *DockerHandler) PullImage(c *gin.Context) {
	var req model.ImagePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	imageRef := req.Image
	if req.Tag != "" {
		imageRef = req.Image + ":" + req.Tag
	} else if !strings.Contains(req.Image, ":") && !strings.Contains(req.Image, "@") {
		imageRef = req.Image + ":latest"
	}

	reader, err := h.dockerClient.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+err.Error())
		return
	}
	defer reader.Close()

	var lastMessage string
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			Status         string `json:"status"`
			Error          string `json:"error"`
			ProgressDetail struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			} `json:"progressDetail"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			helper.RespondError(c, http.StatusInternalServerError, "拉取失败: "+msg.Error)
			return
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pulled", "image", imageRef)
	helper.RespondSuccess(c, "镜像拉取成功", gin.H{"image": imageRef, "status": lastMessage})
}

// ==================== 网络管理 ====================

func (h *DockerHandler) ListNetworks(c *gin.Context) {
	ctx := context.Background()
	networks, err := h.dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		logman.Error("List networks failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取网络列表失败")
		return
	}

	var result []*model.NetworkInfo
	for _, net := range networks {
		subnet := ""
		if len(net.IPAM.Config) > 0 && net.IPAM.Config[0].Subnet != "" {
			subnet = net.IPAM.Config[0].Subnet
		}
		id := net.ID
		if len(id) > 12 {
			id = id[:12]
		}
		result = append(result, &model.NetworkInfo{
			ID: id, Name: net.Name, Driver: net.Driver, Subnet: subnet, Scope: net.Scope,
		})
	}
	helper.RespondSuccess(c, "Networks listed successfully", result)
}

func (h *DockerHandler) NetworkAction(c *gin.Context) {
	var req model.NetworkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	switch req.Action {
	case "remove":
		if err := h.dockerClient.NetworkRemove(ctx, req.ID); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "删除网络失败: "+err.Error())
			return
		}
	default:
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Network action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, "网络操作成功", nil)
}

func (h *DockerHandler) CreateNetwork(c *gin.Context) {
	var req model.NetworkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	driver := req.Driver
	if driver == "" {
		driver = "bridge"
	}

	resp, err := h.dockerClient.NetworkCreate(ctx, req.Name, types.NetworkCreate{Driver: driver})
	if err != nil {
		logman.Error("Create network failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建网络失败: "+err.Error())
		return
	}
	id := resp.ID
	if len(id) > 12 {
		id = id[:12]
	}
	logman.Info("Network created", "name", req.Name, "id", id)
	helper.RespondSuccess(c, "网络创建成功", gin.H{"id": id, "name": req.Name})
}

// ==================== 卷管理 ====================

func (h *DockerHandler) ListVolumes(c *gin.Context) {
	ctx := context.Background()
	volumes, err := h.dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		logman.Error("List volumes failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取卷列表失败")
		return
	}

	var result []*model.VolumeInfo
	for _, vol := range volumes.Volumes {
		result = append(result, &model.VolumeInfo{
			Name: vol.Name, Driver: vol.Driver,
			Mountpoint: vol.Mountpoint, CreatedAt: vol.CreatedAt,
		})
	}
	helper.RespondSuccess(c, "Volumes listed successfully", result)
}

func (h *DockerHandler) VolumeAction(c *gin.Context) {
	var req model.VolumeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	switch req.Action {
	case "remove":
		if err := h.dockerClient.VolumeRemove(ctx, req.Name, true); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "删除卷失败: "+err.Error())
			return
		}
	default:
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Volume action performed", "action", req.Action, "name", req.Name)
	helper.RespondSuccess(c, "卷操作成功", nil)
}

func (h *DockerHandler) CreateVolume(c *gin.Context) {
	var req model.VolumeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()
	driver := req.Driver
	if driver == "" {
		driver = "local"
	}

	resp, err := h.dockerClient.VolumeCreate(ctx, volume.CreateOptions{Name: req.Name, Driver: driver})
	if err != nil {
		logman.Error("Create volume failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建卷失败: "+err.Error())
		return
	}
	logman.Info("Volume created", "name", req.Name)
	helper.RespondSuccess(c, "卷创建成功", gin.H{"name": resp.Name, "mountpoint": resp.Mountpoint})
}

// ==================== 工具函数 ====================

func formatPorts(ports []types.Port) string {
	var result []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			result = append(result, strconv.Itoa(int(p.PublicPort))+"/"+p.Type+"->"+p.IP+":"+strconv.Itoa(int(p.PrivatePort)))
		} else {
			result = append(result, strconv.Itoa(int(p.PrivatePort))+"/"+p.Type)
		}
	}
	return strings.Join(result, ", ")
}

func parseDockerLogs(data []byte) []string {
	var logs []string
	for i := 0; i < len(data); {
		if i+8 > len(data) {
			break
		}
		size := int(data[i+4])<<24 | int(data[i+5])<<16 | int(data[i+6])<<8 | int(data[i+7])
		i += 8
		if i+size > len(data) || size <= 0 {
			break
		}
		logs = append(logs, string(data[i:i+size]))
		i += size
	}
	return logs
}

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
		compose, err = h.autoCreateComposeFile(context.Background(), name)
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

	helper.RespondSuccess(c, "获取容器配置成功", result)
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

	ctx := context.Background()

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

// ==================== 容器终端 ====================

// ContainerExec 容器终端 WebSocket 处理
func (h *DockerHandler) ContainerExec(c *gin.Context) {
	containerID := c.Query("id")
	if containerID == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器ID不能为空")
		return
	}

	shell := c.DefaultQuery("shell", "/bin/sh")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket upgrade error", "error", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	// 创建 exec 实例
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := h.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		h.sendWsMessage(conn, "[创建终端会话失败: "+err.Error()+"]\r\n")
		return
	}

	// 连接到 exec 实例
	attachConfig := types.ExecStartCheck{Tty: true}
	hijackedResp, err := h.dockerClient.ContainerExecAttach(ctx, execResp.ID, attachConfig)
	if err != nil {
		h.sendWsMessage(conn, "[连接终端失败: "+err.Error()+"]\r\n")
		return
	}
	defer hijackedResp.Close()

	h.sendWsMessage(conn, "[容器终端已连接]\r\n")

	// 转发容器输出到 WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					logman.Error("Container exec read error", "error", err)
				}
				return
			}
			if n > 0 {
				h.sendWsMessage(conn, string(buf[:n]))
			}
		}
	}()

	// 转发 WebSocket 输入到容器
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logman.Error("WebSocket read error", "error", err)
			return
		}
		if _, err := hijackedResp.Conn.Write(msg); err != nil {
			logman.Error("Container exec write error", "error", err)
			return
		}
	}
}

// ==================== 容器统计 ====================

// CPU 主频缓存（不频繁变化，5分钟刷新一次）
var (
	cpuFreqCache      float64
	cpuFreqMu         sync.Mutex
	cpuFreqLastUpdate time.Time
)

// getCpuFreq 获取 CPU 主频（使用 gopsutil，跨平台兼容）
func getCpuFreq() float64 {
	cpuFreqMu.Lock()
	defer cpuFreqMu.Unlock()

	// 5分钟内使用缓存
	if time.Since(cpuFreqLastUpdate) < 5*time.Minute && cpuFreqCache > 0 {
		return cpuFreqCache
	}

	cpuFreqCache = 0

	// 使用 gopsutil 获取 CPU 主频
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		cpuFreqCache = cpuInfos[0].Mhz
	}

	cpuFreqLastUpdate = time.Now()
	return cpuFreqCache
}

func (h *DockerHandler) ContainerStats(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "容器ID不能为空")
		return
	}

	ctx := context.Background()

	stats, err := h.dockerClient.ContainerStats(ctx, id, false)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取容器统计信息失败: "+err.Error())
		return
	}
	defer stats.Body.Close()

	data, err := io.ReadAll(stats.Body)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "读取统计信息失败")
		return
	}

	var v types.StatsJSON
	if err := json.Unmarshal(data, &v); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "解析统计信息失败")
		return
	}

	// 从 Docker Stats API 获取 CPU 核心数
	cpuCores := len(v.CPUStats.CPUUsage.PercpuUsage)

	// 计算 CPU 使用率
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(cpuCores) * 100.0
	}

	// 计算内存使用率
	memoryUsage := v.MemoryStats.Usage - v.MemoryStats.Stats["cache"]
	var memoryPercent float64
	if v.MemoryStats.Limit > 0 {
		memoryPercent = float64(memoryUsage) / float64(v.MemoryStats.Limit) * 100.0
	}

	// 计算网络 I/O
	var networkRx, networkTx int64
	networkDetail := make(map[string]*model.NetDetail)
	for name, netStats := range v.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
		networkDetail[name] = &model.NetDetail{
			RxBytes:   netStats.RxBytes,
			RxPackets: netStats.RxPackets,
			RxErrors:  netStats.RxErrors,
			RxDropped: netStats.RxDropped,
			TxBytes:   netStats.TxBytes,
			TxPackets: netStats.TxPackets,
			TxErrors:  netStats.TxErrors,
			TxDropped: netStats.TxDropped,
		}
	}

	// CPU 节流数据
	cpuThrottled := &model.CpuThrottledData{
		Periods:          v.CPUStats.ThrottlingData.Periods,
		ThrottledPeriods: v.CPUStats.ThrottlingData.ThrottledPeriods,
		ThrottledTime:    v.CPUStats.ThrottlingData.ThrottledTime,
	}

	// 获取进程列表
	var processList *model.ContainerProcessList
	topResult, err := h.dockerClient.ContainerTop(ctx, id, nil)
	if err == nil {
		processList = &model.ContainerProcessList{
			Titles:    topResult.Titles,
			Processes: topResult.Processes,
		}
	}

	// 计算磁盘 I/O
	var blockRead, blockWrite int64
	blockDetailMap := make(map[string]*model.BlockDetail)
	for _, blkStats := range v.BlkioStats.IoServiceBytesRecursive {
		switch blkStats.Op {
		case "read":
			blockRead += int64(blkStats.Value)
		case "write":
			blockWrite += int64(blkStats.Value)
		}
		// 按设备聚合详情
		if blkStats.Op == "read" || blkStats.Op == "write" {
			key := fmt.Sprintf("%d:%d", blkStats.Major, blkStats.Minor)
			if _, ok := blockDetailMap[key]; !ok {
				blockDetailMap[key] = &model.BlockDetail{
					Major: blkStats.Major,
					Minor: blkStats.Minor,
				}
			}
			if blkStats.Op == "read" {
				blockDetailMap[key].Read += blkStats.Value
			} else {
				blockDetailMap[key].Write += blkStats.Value
			}
		}
	}
	// 转换为有序列表
	var blockDetail []*model.BlockDetail
	for _, detail := range blockDetailMap {
		blockDetail = append(blockDetail, detail)
	}
	// 按主设备号排序
	sort.Slice(blockDetail, func(i, j int) bool {
		if blockDetail[i].Major != blockDetail[j].Major {
			return blockDetail[i].Major < blockDetail[j].Major
		}
		return blockDetail[i].Minor < blockDetail[j].Minor
	})

	name := ""
	if len(v.Name) > 0 {
		name = strings.TrimPrefix(v.Name, "/")
	}

	// 获取 CPU 主频（使用缓存）
	cpuFreq := getCpuFreq()

	result := model.ContainerStatsResponse{
		ID:            id,
		Name:          name,
		CPUPercent:    math.Round(cpuPercent*100) / 100,
		CpuCores:      cpuCores,
		CpuFreq:       math.Round(cpuFreq*100) / 100,
		MemoryUsage:   int64(memoryUsage),
		MemoryLimit:   int64(v.MemoryStats.Limit),
		MemoryPercent: math.Round(memoryPercent*100) / 100,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		Pids:          int64(v.PidsStats.Current),
		PidsLimit:     int64(v.PidsStats.Limit),
		CpuThrottled:  cpuThrottled,
		NetworkDetail: networkDetail,
		BlockDetail:   blockDetail,
		ProcessList:   processList,
	}

	helper.RespondSuccess(c, "容器统计信息获取成功", result)
}

// ==================== 网络详情 ====================

func (h *DockerHandler) NetworkInspect(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		helper.RespondError(c, http.StatusBadRequest, "网络ID不能为空")
		return
	}

	ctx := context.Background()

	networkInfo, err := h.dockerClient.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取网络详情失败: "+err.Error())
		return
	}

	// 提取已连接的容器信息
	var containers []*model.NetworkContainerInfo
	for endpointID, ep := range networkInfo.Containers {
		name := ep.Name
		if name == "" {
			name = endpointID[:12]
		}
		// 尝试获取容器名称
		containerJSON, err := h.dockerClient.ContainerInspect(ctx, ep.Name)
		if err == nil && len(containerJSON.Name) > 0 {
			name = strings.TrimPrefix(containerJSON.Name, "/")
		}
		containers = append(containers, &model.NetworkContainerInfo{
			ID:         endpointID[:12],
			Name:       name,
			IPv4:       ep.IPv4Address,
			IPv6:       ep.IPv6Address,
			MacAddress: ep.MacAddress,
		})
	}

	result := model.NetworkInspectResponse{
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

	helper.RespondSuccess(c, "网络详情获取成功", result)
}

// ==================== 数据卷详情 ====================

func (h *DockerHandler) VolumeInspect(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.RespondError(c, http.StatusBadRequest, "卷名称不能为空")
		return
	}

	ctx := context.Background()

	volInfo, err := h.dockerClient.VolumeInspect(ctx, name)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "获取卷详情失败: "+err.Error())
		return
	}

	// 查找使用此卷的容器
	var usedBy []*model.VolumeUsedByContainer
	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		for _, ct := range containers {
			for _, mount := range ct.Mounts {
				if mount.Type == "volume" && mount.Name == name {
					ctName := ""
					if len(ct.Names) > 0 {
						ctName = strings.TrimPrefix(ct.Names[0], "/")
					}
					usedBy = append(usedBy, &model.VolumeUsedByContainer{
						ID:        ct.ID[:12],
						Name:      ctName,
						MountPath: mount.Destination,
						ReadOnly:  !mount.RW,
					})
				}
			}
		}
	}

	result := model.VolumeInspectResponse{
		Name:       volInfo.Name,
		Driver:     volInfo.Driver,
		Mountpoint: volInfo.Mountpoint,
		CreatedAt:  volInfo.CreatedAt,
		Scope:      volInfo.Scope,
		UsedBy:     usedBy,
	}

	if volInfo.UsageData != nil {
		result.Size = volInfo.UsageData.Size
		result.RefCount = volInfo.UsageData.RefCount
	}

	helper.RespondSuccess(c, "卷详情获取成功", result)
}

// ==================== 镜像标签 ====================

func (h *DockerHandler) ImageTag(c *gin.Context) {
	var req model.ImageTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()

	if err := h.dockerClient.ImageTag(ctx, req.ID, req.RepoTag); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "镜像打标签失败: "+err.Error())
		return
	}

	logman.Info("Image tagged", "id", req.ID, "tag", req.RepoTag)
	helper.RespondSuccess(c, "镜像标签添加成功", nil)
}

// ==================== 镜像搜索 ====================

func (h *DockerHandler) ImageSearch(c *gin.Context) {
	term := c.Query("term")
	if term == "" {
		helper.RespondError(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	ctx := context.Background()

	results, err := h.dockerClient.ImageSearch(ctx, term, types.ImageSearchOptions{Limit: 25})
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "搜索镜像失败: "+err.Error())
		return
	}

	var searchResults []*model.ImageSearchResult
	for _, r := range results {
		searchResults = append(searchResults, &model.ImageSearchResult{
			Name:        r.Name,
			Description: r.Description,
			IsOfficial:  r.IsOfficial,
			IsAutomated: r.IsAutomated,
			StarCount:   r.StarCount,
		})
	}

	helper.RespondSuccess(c, "搜索完成", searchResults)
}

// ==================== 镜像构建 ====================

func (h *DockerHandler) ImageBuild(c *gin.Context) {
	var req model.ImageBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := context.Background()

	// 构建 Dockerfile 的 tar 包
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	hdr := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(req.Dockerfile)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "构建 Dockerfile 包失败: "+err.Error())
		return
	}
	if _, err := tw.Write([]byte(req.Dockerfile)); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "构建 Dockerfile 包失败: "+err.Error())
		return
	}
	tw.Close()

	tag := req.Tag
	if tag == "" {
		tag = "custom:latest"
	}

	resp, err := h.dockerClient.ImageBuild(ctx, tarBuf, types.ImageBuildOptions{
		Tags: []string{tag},
	})
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "构建镜像失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	var lastMessage string
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			helper.RespondError(c, http.StatusInternalServerError, "构建失败: "+msg.Error)
			return
		}
		if msg.Stream != "" {
			lastMessage = strings.TrimSpace(msg.Stream)
		}
	}

	logman.Info("Image built", "tag", tag)
	helper.RespondSuccess(c, "镜像构建成功", gin.H{"tag": tag, "status": lastMessage})
}

// sendWsMessage 发送消息到 WebSocket
func (h *DockerHandler) sendWsMessage(conn *websocket.Conn, msg string) {
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
}
