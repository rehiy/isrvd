package docker

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
	"github.com/shirou/gopsutil/v3/cpu"

	pkgDocker "isrvd/pkgs/docker"
	"isrvd/internal/service/wsterm"
)

var (
	cpuFreqCache      float64
	cpuFreqMu         sync.Mutex
	cpuFreqLastUpdate time.Time
)

// ContainerInfo Docker 容器信息（列表项），保持前端稳定响应结构。
type ContainerInfo struct {
	ID       string            `json:"id"`                 // 容器 ID
	Name     string            `json:"name"`               // 容器名称
	Image    string            `json:"image"`              // 镜像名称
	State    string            `json:"state"`              // 运行状态（running/exited/paused）
	Status   string            `json:"status"`             // 状态描述
	Ports    []string          `json:"ports"`              // 端口映射列表
	Networks []string          `json:"networks,omitempty"` // 所属网络
	Created  int64             `json:"created"`            // 创建时间戳
	IsSwarm  bool              `json:"isSwarm,omitempty"`  // 是否为 Swarm 管理的容器
	IsSelf   bool              `json:"isSelf,omitempty"`   // 是否为当前服务容器
	Labels   map[string]string `json:"labels,omitempty"`   // 容器标签
}

// ContainerList 列出容器
func (s *Service) ContainerList(ctx context.Context, all bool) ([]*ContainerInfo, error) {
	list, err := s.docker.ContainerList(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}
	selfID := s.docker.SelfContainerID(ctx)
	result := make([]*ContainerInfo, 0, len(list))
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		networks := make([]string, 0)
		if ct.NetworkSettings != nil {
			for netName := range ct.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}
		result = append(result, &ContainerInfo{
			ID:       pkgDocker.ShortID(ct.ID),
			Name:     name,
			Image:    ct.Image,
			State:    string(ct.State),
			Status:   ct.Status,
			Ports:    formatPorts(ct.Ports),
			Networks: networks,
			Created:  ct.Created,
			IsSwarm:  ct.Labels["com.docker.swarm.service.id"] != "",
			IsSelf:   selfID != "" && ct.ID == selfID,
			Labels:   ct.Labels,
		})
	}
	return result, nil
}

// VolumeMapping 挂载映射，保持 HTTP API 兼容。
type VolumeMapping struct {
	Type          string `json:"type,omitempty"`     // 挂载类型（volume/bind）
	Source        string `json:"source,omitempty"`   // 卷名（volume 类型）
	HostPath      string `json:"hostPath,omitempty"` // 宿主机路径（bind 类型）
	ContainerPath string `json:"containerPath"`      // 容器内挂载路径
	ReadOnly      bool   `json:"readOnly"`           // 是否只读
}

// ContainerSpec 容器可写配置（创建/更新共用），保持 HTTP API 兼容。
type ContainerSpec struct {
	Image      string            `json:"image" binding:"required"` // 镜像名称（含 tag）
	Name       string            `json:"name" binding:"required"`  // 容器名称
	Cmd        []string          `json:"cmd"`                      // 容器启动命令
	Env        []string          `json:"env"`                      // 环境变量列表（格式：KEY=VALUE）
	Ports      map[string]string `json:"ports"`                    // 端口映射（格式：容器端口:主机端口）
	Volumes    []VolumeMapping   `json:"volumes"`                  // 挂载卷列表
	Network    string            `json:"network"`                  // 网络模式
	Restart    string            `json:"restart"`                  // 重启策略（no/on-failure/always/unless-stopped）
	Memory     int64             `json:"memory"`                   // 内存限制（字节）
	Cpus       float64           `json:"cpus"`                     // CPU 核数限制
	Workdir    string            `json:"workdir"`                  // 工作目录
	User       string            `json:"user"`                     // 运行用户
	Hostname   string            `json:"hostname"`                 // 主机名
	Privileged bool              `json:"privileged"`               // 是否特权模式
	CapAdd     []string          `json:"capAdd"`                   // 添加的内核能力
	CapDrop    []string          `json:"capDrop"`                  // 删除的内核能力
	AutoRemove bool              `json:"autoRemove"`               // 退出时自动删除容器
	Labels     map[string]string `json:"labels,omitempty"`         // 标签列表
}

// ContainerDetail 容器详情，保持前端稳定响应结构。
type ContainerDetail struct {
	ContainerSpec
	ID        string            `json:"id"`               // 容器 ID
	Name      string            `json:"name"`             // 容器名称
	State     string            `json:"state"`            // 运行状态
	CreatedAt string            `json:"createdAt"`        // 创建时间
	Labels    map[string]string `json:"labels,omitempty"` // 容器标签
}

// ContainerInspect 获取容器详情
func (s *Service) ContainerInspect(ctx context.Context, id string) (*ContainerDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	info, err := s.docker.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器详情失败: %w", err)
	}
	if info.Config == nil || info.HostConfig == nil || info.State == nil {
		return nil, fmt.Errorf("容器数据不完整")
	}
	ports := make(map[string]string)
	for containerPort, bindings := range info.HostConfig.PortBindings {
		for _, b := range bindings {
			hostPort := b.HostPort
			if b.HostIP != "" && b.HostIP != "0.0.0.0" {
				hostPort = b.HostIP + ":" + hostPort
			}
			ports[hostPort+"/"+containerPort.Proto()] = containerPort.Port()
		}
	}
	volumes := make([]VolumeMapping, 0, len(info.Mounts))
	for _, m := range info.Mounts {
		volumes = append(volumes, VolumeMapping{
			Type:          string(m.Type),
			Source:        m.Source,
			ContainerPath: m.Destination,
			ReadOnly:      !m.RW,
		})
	}
	return &ContainerDetail{
		ContainerSpec: ContainerSpec{
			Image:      info.Config.Image,
			Cmd:        info.Config.Cmd,
			Env:        info.Config.Env,
			Ports:      ports,
			Volumes:    volumes,
			Network:    string(info.HostConfig.NetworkMode),
			Restart:    string(info.HostConfig.RestartPolicy.Name),
			Memory:     info.HostConfig.Memory / (1024 * 1024),
			Cpus:       float64(info.HostConfig.NanoCPUs) / 1e9,
			Workdir:    info.Config.WorkingDir,
			User:       info.Config.User,
			Hostname:   info.Config.Hostname,
			Privileged: info.HostConfig.Privileged,
			CapAdd:     []string(info.HostConfig.CapAdd),
			CapDrop:    []string(info.HostConfig.CapDrop),
		},
		ID:        pkgDocker.ShortID(info.ID),
		Name:      strings.TrimPrefix(info.Name, "/"),
		State:     info.State.Status,
		CreatedAt: info.Created,
		Labels:    info.Config.Labels,
	}, nil
}

// ContainerCreateResult 创建容器结果
type ContainerCreateResult struct {
	ID   string `json:"id"`   // 新建容器 ID
	Name string `json:"name"` // 新建容器名称
}

// ContainerCreate 创建并启动容器。
func (s *Service) ContainerCreate(ctx context.Context, req ContainerSpec) (*ContainerCreateResult, error) {
	if req.Image == "" {
		return nil, fmt.Errorf("镜像名称不能为空")
	}
	if err := s.docker.ImageEnsure(ctx, req.Image, false); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", req.Image, err)
	}
	containerConfig, hostConfig, err := s.containerCreateConfig(req)
	if err != nil {
		return nil, err
	}
	id, err := s.docker.ContainerCreateAndStart(ctx, req.Name, containerConfig, hostConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	return &ContainerCreateResult{ID: pkgDocker.ShortID(id), Name: req.Name}, nil
}

// ContainerCreateRaw 使用 Docker SDK 原始结构创建容器，供 Compose 部署链路复用。
func (s *Service) ContainerCreateRaw(ctx context.Context, name string, containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) (*ContainerCreateResult, error) {
	if containerConfig == nil || containerConfig.Image == "" {
		return nil, fmt.Errorf("镜像名称不能为空")
	}
	if err := s.docker.ImageEnsure(ctx, containerConfig.Image, false); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", containerConfig.Image, err)
	}
	id, err := s.docker.ContainerCreateAndStart(ctx, name, containerConfig, hostConfig, networkingConfig)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	return &ContainerCreateResult{ID: pkgDocker.ShortID(id), Name: name}, nil
}

// ContainerStatsResponse 容器统计信息响应，保持前端稳定响应结构。
type ContainerStatsResponse struct {
	ID            string                     `json:"id"`            // 容器 ID
	Name          string                     `json:"name"`          // 容器名称
	CPUPercent    float64                    `json:"cpuPercent"`    // CPU 使用率（百分比）
	CPUCores      int                        `json:"cpuCores"`      // 可用 CPU 核数
	CPUFreq       float64                    `json:"cpuFreq"`       // CPU 主频（MHz）
	MemoryUsage   int64                      `json:"memoryUsage"`   // 内存使用量（字节）
	MemoryLimit   int64                      `json:"memoryLimit"`   // 内存上限（字节）
	MemoryPercent float64                    `json:"memoryPercent"` // 内存使用率（百分比）
	NetworkRx     int64                      `json:"networkRx"`     // 网络接收总量（字节）
	NetworkTx     int64                      `json:"networkTx"`     // 网络发送总量（字节）
	BlockRead     int64                      `json:"blockRead"`     // 磁盘读取总量（字节）
	BlockWrite    int64                      `json:"blockWrite"`    // 磁盘写入总量（字节）
	Pids          int64                      `json:"pids"`          // 当前进程数
	PidsLimit     int64                      `json:"pidsLimit"`     // 进程数上限
	CPUThrottled  *CPUThrottledData          `json:"cpuThrottled"`  // CPU 节流明细
	NetworkDetail map[string]*StatsNetDetail `json:"networkDetail"` // 各网卡明细（按网卡名）
	BlockDetail   []*StatsBlockDetail        `json:"blockDetail"`   // 各块设备明细
	ProcessList   *ContainerProcessList      `json:"processList"`   // 容器内进程列表
}

// CPUThrottledData CPU 节流数据，保持前端稳定响应结构。
type CPUThrottledData struct {
	Periods          uint64 `json:"periods"`          // 调度周期总数
	ThrottledPeriods uint64 `json:"throttledPeriods"` // 被节流的周期数
	ThrottledTime    uint64 `json:"throttledTime"`    // 被节流的累计时间（纳秒）
}

// StatsNetDetail 网卡详细统计，保持前端稳定响应结构。
type StatsNetDetail struct {
	RxBytes   uint64 `json:"rxBytes"`   // 接收字节数
	RxPackets uint64 `json:"rxPackets"` // 接收包数
	RxErrors  uint64 `json:"rxErrors"`  // 接收错误数
	RxDropped uint64 `json:"rxDropped"` // 接收丢包数
	TxBytes   uint64 `json:"txBytes"`   // 发送字节数
	TxPackets uint64 `json:"txPackets"` // 发送包数
	TxErrors  uint64 `json:"txErrors"`  // 发送错误数
	TxDropped uint64 `json:"txDropped"` // 发送丢包数
}

// StatsBlockDetail 硬盘设备详细统计，保持前端稳定响应结构。
type StatsBlockDetail struct {
	Major uint64 `json:"major"` // 设备主号
	Minor uint64 `json:"minor"` // 设备次号
	Read  uint64 `json:"read"`  // 读取字节数
	Write uint64 `json:"write"` // 写入字节数
}

// ContainerProcessList 容器进程列表，保持前端稳定响应结构。
type ContainerProcessList struct {
	Titles    []string   `json:"titles"`    // 列标题（如 PID/USER/CMD）
	Processes [][]string `json:"processes"` // 进程行数据，每行与 titles 对应
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (*ContainerStatsResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	stats, top, err := s.docker.ContainerStats(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器统计信息失败: %w", err)
	}
	return containerStatsResponse(id, stats, top), nil
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("容器ID不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.ContainerAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("容器操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// ContainerLogsRequest 日志请求，保持 HTTP API 兼容。
type ContainerLogsRequest struct {
	ID   string `json:"id" binding:"required"` // 容器 ID
	Tail string `json:"tail"`                  // 返回末尾行数（如 "100"，空表示全部）
}

// ContainerLogsResult 容器日志结果，保持前端稳定响应结构。
type ContainerLogsResult struct {
	ID   string   `json:"id"`   // 容器 ID
	Logs []string `json:"logs"` // 日志行列表
}

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req ContainerLogsRequest) (*ContainerLogsResult, error) {
	if req.ID == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	logs, err := s.docker.ContainerLogs(ctx, req.ID, req.Tail)
	if err != nil {
		return nil, fmt.Errorf("获取容器日志失败: %w", err)
	}
	return &ContainerLogsResult{ID: req.ID, Logs: logs}, nil
}

// ContainerLogsStream 容器实时日志流
func (s *Service) ContainerLogsStream(ctx context.Context, w io.Writer, req ContainerLogsRequest) {
	s.docker.ContainerLogsStream(ctx, w, req.ID, req.Tail)
}

// ContainerExec 容器终端 WebSocket 桥接：建立 exec 会话后与 WebSocket 双向转发
func (s *Service) ContainerExec(ctx context.Context, conn *websocket.ServerConn, containerID, shell string) {
	session, err := s.docker.ContainerExecAttach(ctx, containerID, shell)
	if err != nil {
		conn.Write([]byte("[" + err.Error() + "]\r\n"))
		return
	}

	conn.Write([]byte("[容器终端已连接]\r\n"))

	// 终端 WSS 保活心跳，避免空闲被中间层断开
	stop := wsterm.KeepAlive(conn, wsterm.HeartbeatInterval)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := session.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					logman.Error("container exec read error", "error", err)
				}
				return
			}
		}
	}()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logman.Error("websocket read error", "error", err)
			break
		}
		if n > 0 {
			if _, err := session.Write(buf[:n]); err != nil {
				logman.Error("container exec write error", "error", err)
				break
			}
		}
	}

	session.Close()
	<-done
}

// Info 获取 Docker 概览信息
func (s *Service) Info(ctx context.Context) (*DockerInfo, error) {
	daemonInfo, err := s.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Docker 信息失败: %w", err)
	}

	containers, err := s.docker.ContainerList(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
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

	images, err := s.docker.ImageList(ctx, true)
	if err != nil {
		logman.Warn("ImageList failed", "error", err)
	}
	volumes, err := s.docker.VolumeList(ctx)
	if err != nil {
		logman.Warn("VolumeList failed", "error", err)
	}
	networks, err := s.docker.NetworkList(ctx)
	if err != nil {
		logman.Warn("NetworkList failed", "error", err)
	}

	var mirrors []string
	if daemonInfo.RegistryConfig != nil {
		mirrors = daemonInfo.RegistryConfig.Mirrors
	}

	return &DockerInfo{
		ContainersRunning:  running,
		ContainersStopped:  stopped,
		ContainersPaused:   paused,
		ImagesTotal:        int64(len(images)),
		VolumesTotal:       int64(len(volumes)),
		NetworksTotal:      int64(len(networks)),
		RegistryMirrors:    mirrors,
		IndexServerAddress: daemonInfo.IndexServerAddress,
	}, nil
}

// ─── 内部方法 ───

func (s *Service) containerCreateConfig(req ContainerSpec) (*container.Config, *container.HostConfig, error) {
	containerConfig := &container.Config{
		Image:      req.Image,
		Cmd:        req.Cmd,
		Env:        req.Env,
		WorkingDir: req.Workdir,
		User:       req.User,
		Hostname:   req.Hostname,
		Labels:     req.Labels,
	}
	hostConfig := &container.HostConfig{}
	switch req.Restart {
	case "always", "on-failure", "unless-stopped":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.Restart)}
	default:
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "no"}
	}
	if req.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(req.Network)
	}
	if req.Memory > 0 {
		hostConfig.Memory = req.Memory * 1024 * 1024
	}
	if req.Cpus > 0 {
		hostConfig.NanoCPUs = int64(req.Cpus * 1e9)
	}
	if len(req.Ports) > 0 {
		portBindings := make(nat.PortMap)
		exposedPorts := make(nat.PortSet)
		for hostPortSpec, containerPort := range req.Ports {
			hostPort := hostPortSpec
			hostIP := "0.0.0.0"
			proto := "tcp"
			if idx := strings.LastIndex(hostPortSpec, "/"); idx >= 0 {
				hostPort = hostPortSpec[:idx]
				proto = hostPortSpec[idx+1:]
			}
			if idx := strings.LastIndex(hostPort, ":"); idx >= 0 {
				hostIP = hostPort[:idx]
				hostPort = hostPort[idx+1:]
			}
			if idx := strings.Index(containerPort, "/"); idx >= 0 {
				containerPort = containerPort[:idx]
			}
			port := nat.Port(containerPort + "/" + proto)
			portBindings[port] = []nat.PortBinding{{HostIP: hostIP, HostPort: hostPort}}
			exposedPorts[port] = struct{}{}
		}
		hostConfig.PortBindings = portBindings
		containerConfig.ExposedPorts = exposedPorts
	}
	for _, vol := range req.Volumes {
		m, err := s.buildMount(req.Name, vol)
		if err != nil {
			return nil, nil, err
		}
		hostConfig.Mounts = append(hostConfig.Mounts, m)
	}
	hostConfig.Privileged = req.Privileged
	hostConfig.CapAdd = req.CapAdd
	hostConfig.CapDrop = req.CapDrop
	hostConfig.AutoRemove = req.AutoRemove
	return containerConfig, hostConfig, nil
}

func (s *Service) buildMount(containerName string, vol VolumeMapping) (mount.Mount, error) {
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
		if strings.ContainsRune(source, '/') {
			return mount.Mount{}, fmt.Errorf("volume 名称不能包含路径分隔符，请使用 bind 类型或改用合法的 volume 名: %s", source)
		}
		return mount.Mount{Type: mount.TypeVolume, Source: source, Target: vol.ContainerPath, ReadOnly: vol.ReadOnly}, nil
	case string(mount.TypeBind):
		bindSource, err := s.resolveBindSource(containerName, source)
		if err != nil {
			return mount.Mount{}, err
		}
		return mount.Mount{Type: mount.TypeBind, Source: bindSource, Target: vol.ContainerPath, ReadOnly: vol.ReadOnly, BindOptions: &mount.BindOptions{CreateMountpoint: true}}, nil
	default:
		return mount.Mount{}, fmt.Errorf("不支持的挂载类型: %s", mountType)
	}
}

func (s *Service) resolveBindSource(containerName, source string) (string, error) {
	bindSource := source
	if root := s.docker.ContainerRoot(); root != "" && !filepath.IsAbs(bindSource) {
		bindSource = filepath.Join(root, containerName, bindSource)
	}
	if _, err := os.Stat(bindSource); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("检查挂载源失败: %w", err)
	}
	return bindSource, nil
}

func containerStatsResponse(id string, v container.StatsResponse, top *container.TopResponse) *ContainerStatsResponse {
	cpuCores := int(v.CPUStats.OnlineCPUs)
	if cpuCores == 0 {
		cpuCores = len(v.CPUStats.CPUUsage.PercpuUsage)
	}
	if cpuCores == 0 {
		cpuCores = 1
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(cpuCores) * 100.0
	}

	memCache := v.MemoryStats.Stats["inactive_file"]
	if memCache == 0 {
		memCache = v.MemoryStats.Stats["cache"]
	}
	memoryUsage := v.MemoryStats.Usage - memCache
	var memoryPercent float64
	if v.MemoryStats.Limit > 0 {
		memoryPercent = float64(memoryUsage) / float64(v.MemoryStats.Limit) * 100.0
	}

	var networkRx, networkTx int64
	networkDetail := make(map[string]*StatsNetDetail)
	for name, netStats := range v.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
		networkDetail[name] = &StatsNetDetail{
			RxBytes: netStats.RxBytes, RxPackets: netStats.RxPackets, RxErrors: netStats.RxErrors, RxDropped: netStats.RxDropped,
			TxBytes: netStats.TxBytes, TxPackets: netStats.TxPackets, TxErrors: netStats.TxErrors, TxDropped: netStats.TxDropped,
		}
	}

	var blockRead, blockWrite int64
	blockDetailMap := make(map[string]*StatsBlockDetail)
	for _, blkStats := range v.BlkioStats.IoServiceBytesRecursive {
		switch blkStats.Op {
		case "read":
			blockRead += int64(blkStats.Value)
		case "write":
			blockWrite += int64(blkStats.Value)
		}
		if blkStats.Op == "read" || blkStats.Op == "write" {
			key := fmt.Sprintf("%d:%d", blkStats.Major, blkStats.Minor)
			if _, ok := blockDetailMap[key]; !ok {
				blockDetailMap[key] = &StatsBlockDetail{Major: blkStats.Major, Minor: blkStats.Minor}
			}
			if blkStats.Op == "read" {
				blockDetailMap[key].Read += blkStats.Value
			} else {
				blockDetailMap[key].Write += blkStats.Value
			}
		}
	}
	blockDetail := make([]*StatsBlockDetail, 0, len(blockDetailMap))
	for _, detail := range blockDetailMap {
		blockDetail = append(blockDetail, detail)
	}
	sort.Slice(blockDetail, func(i, j int) bool {
		if blockDetail[i].Major != blockDetail[j].Major {
			return blockDetail[i].Major < blockDetail[j].Major
		}
		return blockDetail[i].Minor < blockDetail[j].Minor
	})

	name := strings.TrimPrefix(v.Name, "/")
	var processList *ContainerProcessList
	if top != nil {
		processList = &ContainerProcessList{Titles: top.Titles, Processes: top.Processes}
	}

	return &ContainerStatsResponse{
		ID: id, Name: name, CPUPercent: math.Round(cpuPercent*100) / 100, CPUCores: cpuCores, CPUFreq: math.Round(getCPUFreq()*100) / 100,
		MemoryUsage: int64(memoryUsage), MemoryLimit: int64(v.MemoryStats.Limit), MemoryPercent: math.Round(memoryPercent*100) / 100,
		NetworkRx: networkRx, NetworkTx: networkTx, BlockRead: blockRead, BlockWrite: blockWrite,
		Pids: int64(v.PidsStats.Current), PidsLimit: int64(v.PidsStats.Limit),
		CPUThrottled:  &CPUThrottledData{Periods: v.CPUStats.ThrottlingData.Periods, ThrottledPeriods: v.CPUStats.ThrottlingData.ThrottledPeriods, ThrottledTime: v.CPUStats.ThrottlingData.ThrottledTime},
		NetworkDetail: networkDetail, BlockDetail: blockDetail, ProcessList: processList,
	}
}

// ─── 辅助函数 ───

func formatPorts(ports []container.Port) []string {
	seen := make(map[string]bool, len(ports))
	result := make([]string, 0, len(ports))
	for _, p := range ports {
		var entry, key string
		isIPv6 := strings.Contains(p.IP, ":")
		if p.PublicPort > 0 {
			key = fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			if isIPv6 && seen[key] {
				continue
			}
			if p.IP == "" || p.IP == "0.0.0.0" || p.IP == "::" {
				entry = fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			} else {
				entry = fmt.Sprintf("%s:%d:%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
			}
		} else {
			key = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
			entry = key
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, entry)
		}
	}
	return result
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

func getCPUFreq() float64 {
	cpuFreqMu.Lock()
	defer cpuFreqMu.Unlock()
	if time.Since(cpuFreqLastUpdate) < 5*time.Minute && cpuFreqCache > 0 {
		return cpuFreqCache
	}
	cpuFreqCache = 0
	if cpuInfos, err := cpu.Info(); err == nil && len(cpuInfos) > 0 {
		cpuFreqCache = cpuInfos[0].Mhz
	}
	cpuFreqLastUpdate = time.Now()
	return cpuFreqCache
}
