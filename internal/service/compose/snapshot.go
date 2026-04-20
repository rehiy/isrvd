// Package compose 业务层：组合 pkgs/compose 通用能力 + 项目约定路径。
package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

const composeFileName = "docker-compose.yml"

// SnapshotService 容器 compose 快照服务
//
// 业务约定：为每个用户创建的容器，在 {ContainerRoot}/{name}/docker-compose.yml
// 生成一份 compose 快照，便于外部运维工具直接 `docker compose up` 重建。
type SnapshotService struct {
	docker *docker.DockerService
}

// NewSnapshotService 创建快照服务
func NewSnapshotService(d *docker.DockerService) (*SnapshotService, error) {
	if d == nil {
		return nil, fmt.Errorf("docker service 未提供")
	}
	return &SnapshotService{docker: d}, nil
}

// ContainerConfigResponse 容器配置响应（与前端约定）
type ContainerConfigResponse struct {
	Image      string                 `json:"image"`
	Name       string                 `json:"name"`
	Cmd        []string               `json:"cmd,omitempty"`
	Env        []string               `json:"env,omitempty"`
	Ports      map[string]string      `json:"ports,omitempty"`
	Volumes    []docker.VolumeMapping `json:"volumes,omitempty"`
	Network    string                 `json:"network,omitempty"`
	Restart    string                 `json:"restart,omitempty"`
	Memory     int64                  `json:"memory,omitempty"`
	Cpus       float64                `json:"cpus,omitempty"`
	Workdir    string                 `json:"workdir,omitempty"`
	User       string                 `json:"user,omitempty"`
	Hostname   string                 `json:"hostname,omitempty"`
	Privileged bool                   `json:"privileged,omitempty"`
	CapAdd     []string               `json:"capAdd,omitempty"`
	CapDrop    []string               `json:"capDrop,omitempty"`
}

// Save 根据 ContainerCreateRequest 生成并持久化 compose 快照（失败仅记日志）
func (s *SnapshotService) Save(req docker.ContainerCreateRequest) {
	path := s.path(req.Name)
	if path == "" {
		return
	}
	project, err := compose.ProjectFromCreateRequest(req)
	if err != nil {
		logman.Warn("Build compose snapshot failed", "name", req.Name, "error", err)
		return
	}
	s.writeSnapshot(path, project, req.Name)
}

// SaveFromContainer 从运行态容器反推 compose 快照（用于老容器补齐或手动刷新）
func (s *SnapshotService) SaveFromContainer(ctx context.Context, idOrName string) error {
	info, err := s.docker.InspectContainer(ctx, idOrName)
	if err != nil {
		return fmt.Errorf("读取容器运行态失败: %w", err)
	}
	project, err := compose.ProjectFromInspect(info)
	if err != nil {
		return err
	}
	path := s.path(project.Name)
	if path == "" {
		return fmt.Errorf("容器根目录未配置，无法写入快照")
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建快照目录失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入 compose 快照失败: %w", err)
	}
	logman.Info("Compose snapshot saved from container", "name", project.Name, "path", path)
	return nil
}

// writeSnapshot 将 project 序列化并写入 path（失败仅记日志）
func (s *SnapshotService) writeSnapshot(path string, project *types.Project, name string) {
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		logman.Warn("Marshal compose snapshot failed", "name", name, "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		logman.Warn("Create snapshot dir failed", "name", name, "error", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		logman.Warn("Write compose snapshot failed", "name", name, "error", err)
	}
}

// GetContainerConfig 从 compose 快照读取容器配置；快照缺失时自动从运行态反推
func (s *SnapshotService) GetContainerConfig(ctx context.Context, name string) (*ContainerConfigResponse, error) {
	path := s.path(name)
	if path == "" {
		return nil, fmt.Errorf("容器名称或根目录未配置")
	}

	// 快照缺失：从运行态 inspect 反推并写入
	if _, err := os.Stat(path); err != nil {
		info, ierr := s.docker.InspectContainer(ctx, name)
		if ierr != nil {
			return nil, fmt.Errorf("compose 快照不存在且读取运行态失败: %w", ierr)
		}
		project, perr := compose.ProjectFromInspect(info)
		if perr != nil {
			return nil, perr
		}
		s.writeSnapshot(path, project, name)
		if svc, ok := firstService(project, name); ok {
			return serviceToResponse(name, svc), nil
		}
		return nil, fmt.Errorf("反推配置失败：project 中没有服务")
	}

	project, err := compose.LoadProject(ctx, compose.LoadOptions{
		WorkingDir:  filepath.Dir(path),
		ConfigFiles: []string{path},
		ProjectName: name,
	})
	if err != nil {
		return nil, fmt.Errorf("解析 compose 文件失败: %w", err)
	}

	svc, ok := firstService(project, name)
	if !ok {
		return nil, fmt.Errorf("配置文件中未找到该容器")
	}
	return serviceToResponse(name, svc), nil
}

// firstService 按 name 匹配服务，找不到时返回第一个（兼容 key 与容器名不一致）
func firstService(project *types.Project, name string) (types.ServiceConfig, bool) {
	if svc, ok := project.Services[name]; ok {
		return svc, true
	}
	for _, v := range project.Services {
		return v, true
	}
	return types.ServiceConfig{}, false
}

// path 返回容器对应的 compose 快照完整路径；未配置根目录或名称为空时返回 ""
func (s *SnapshotService) path(name string) string {
	root := s.docker.ContainerRoot()
	if root == "" || name == "" {
		return ""
	}
	return filepath.Join(root, name, composeFileName)
}

// serviceToResponse 将 ServiceConfig 转为前端响应 DTO
func serviceToResponse(name string, svc types.ServiceConfig) *ContainerConfigResponse {
	resp := &ContainerConfigResponse{
		Image:      svc.Image,
		Name:       name,
		Cmd:        []string(svc.Command),
		Env:        envToSlice(svc.Environment),
		Network:    svc.NetworkMode,
		Restart:    svc.Restart,
		Workdir:    svc.WorkingDir,
		User:       svc.User,
		Hostname:   svc.Hostname,
		Privileged: svc.Privileged,
		CapAdd:     svc.CapAdd,
		CapDrop:    svc.CapDrop,
	}

	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		if resp.Ports == nil {
			resp.Ports = map[string]string{}
		}
		host := p.Published
		if host == "" {
			host = strconv.FormatUint(uint64(p.Target), 10)
		}
		resp.Ports[host] = strconv.FormatUint(uint64(p.Target), 10)
	}

	for _, v := range svc.Volumes {
		if v.Source == "" || v.Target == "" {
			continue
		}
		resp.Volumes = append(resp.Volumes, docker.VolumeMapping{
			HostPath:      v.Source,
			ContainerPath: v.Target,
			ReadOnly:      v.ReadOnly,
		})
	}

	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		lim := svc.Deploy.Resources.Limits
		resp.Memory = int64(lim.MemoryBytes) / (1024 * 1024)
		resp.Cpus = float64(lim.NanoCPUs)
	}
	if resp.Memory == 0 && svc.MemLimit > 0 {
		resp.Memory = int64(svc.MemLimit) / (1024 * 1024)
	}
	if resp.Cpus == 0 && svc.CPUS > 0 {
		resp.Cpus = float64(svc.CPUS)
	}
	return resp
}

// envToSlice 将 MappingWithEquals 转成 KEY=VALUE 列表
func envToSlice(env types.MappingWithEquals) []string {
	if len(env) == 0 {
		return nil
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		if v == nil {
			continue
		}
		result = append(result, k+"="+*v)
	}
	return result
}
