package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

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

	resp, err := h.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, req.Name)
	if err != nil {
		logman.Error("Create container failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建容器失败: "+err.Error())
		return
	}

	h.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})

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
