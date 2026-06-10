package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"

	pkgdocker "isrvd/pkgs/docker"
)

// ContainerInfo Docker 容器信息（列表项），保持前端稳定响应结构。
type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	State    string            `json:"state"`
	Status   string            `json:"status"`
	Ports    []string          `json:"ports"`
	Networks []string          `json:"networks,omitempty"`
	Created  int64             `json:"created"`
	IsSwarm  bool              `json:"isSwarm,omitempty"`
	IsSelf   bool              `json:"isSelf,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// ContainerDetail 容器详情，保持前端稳定响应结构。
type ContainerDetail struct {
	ContainerSpec
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     string            `json:"state"`
	CreatedAt string            `json:"createdAt"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ContainerSpec 容器可写配置（创建/更新共用），保持 HTTP API 兼容。
type ContainerSpec struct {
	Image      string                    `json:"image" binding:"required"`
	Name       string                    `json:"name" binding:"required"`
	Cmd        []string                  `json:"cmd"`
	Env        []string                  `json:"env"`
	Ports      map[string]string         `json:"ports"`
	Volumes    []pkgdocker.VolumeMapping `json:"volumes"`
	Network    string                    `json:"network"`
	Restart    string                    `json:"restart"`
	Memory     int64                     `json:"memory"`
	Cpus       float64                   `json:"cpus"`
	Workdir    string                    `json:"workdir"`
	User       string                    `json:"user"`
	Hostname   string                    `json:"hostname"`
	Privileged bool                      `json:"privileged"`
	CapAdd     []string                  `json:"capAdd"`
	CapDrop    []string                  `json:"capDrop"`
	AutoRemove bool                      `json:"autoRemove"`
	Labels     map[string]string         `json:"labels,omitempty"`
}

// ContainerCreateResult 创建容器结果
type ContainerCreateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContainerList 列出容器
func (s *Service) ContainerList(ctx context.Context, all bool) ([]*ContainerInfo, error) {
	list, err := s.docker.ContainerList(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}
	selfID := s.docker.GetSelfContainerID(ctx)
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
			ID:       pkgdocker.ShortID(ct.ID),
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
	volumes := make([]pkgdocker.VolumeMapping, 0, len(info.Mounts))
	for _, m := range info.Mounts {
		volumes = append(volumes, pkgdocker.VolumeMapping{
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
		ID:        pkgdocker.ShortID(info.ID),
		Name:      strings.TrimPrefix(info.Name, "/"),
		State:     info.State.Status,
		CreatedAt: info.Created,
		Labels:    info.Config.Labels,
	}, nil
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
	id, err := s.docker.ContainerCreate(ctx, req.Name, containerConfig, hostConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	return &ContainerCreateResult{ID: pkgdocker.ShortID(id), Name: req.Name}, nil
}

// ContainerCreateRaw 使用 Docker SDK 原始结构创建容器，供 Compose 部署链路复用。
func (s *Service) ContainerCreateRaw(ctx context.Context, name string, containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) (*ContainerCreateResult, error) {
	if containerConfig == nil || containerConfig.Image == "" {
		return nil, fmt.Errorf("镜像名称不能为空")
	}
	if err := s.docker.ImageEnsure(ctx, containerConfig.Image, false); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", containerConfig.Image, err)
	}
	id, err := s.docker.ContainerCreate(ctx, name, containerConfig, hostConfig, networkingConfig)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	return &ContainerCreateResult{ID: pkgdocker.ShortID(id), Name: name}, nil
}

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

func (s *Service) buildMount(containerName string, vol pkgdocker.VolumeMapping) (mount.Mount, error) {
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

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (*pkgdocker.ContainerStatsResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	stats, err := s.docker.ContainerStats(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器统计信息失败: %w", err)
	}
	return stats, nil
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req pkgdocker.ActionRequest) error {
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

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req pkgdocker.ContainerLogsRequest) (*pkgdocker.ContainerLogsResult, error) {
	if req.ID == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	result, err := s.docker.ContainerLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取容器日志失败: %w", err)
	}
	return result, nil
}

// ContainerLogsStream 容器实时日志流
func (s *Service) ContainerLogsStream(ctx context.Context, w io.Writer, req pkgdocker.ContainerLogsRequest) {
	s.docker.ContainerLogsStream(ctx, w, req)
}

// ContainerExec 容器终端 WebSocket 桥接：建立 exec 会话后与 WebSocket 双向转发
func (s *Service) ContainerExec(ctx context.Context, conn *websocket.ServerConn, containerID, shell string) {
	session, err := s.docker.ContainerExecAttach(ctx, containerID, shell)
	if err != nil {
		conn.Write([]byte("[" + err.Error() + "]\r\n"))
		return
	}

	conn.Write([]byte("[容器终端已连接]\r\n"))

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
func (s *Service) Info(ctx context.Context) (*pkgdocker.DockerInfo, error) {
	info, err := s.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Docker 信息失败: %w", err)
	}
	return info, nil
}
