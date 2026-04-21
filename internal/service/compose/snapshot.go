package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

// ensureSnapshot 确保快照文件存在；缺失时从运行态反推并写入。
// 返回内存中的 project（供调用方直接使用，避免二次解析）；快照已存在时返回 nil。
func (s *SnapshotService) ensureSnapshot(ctx context.Context, name, path string) (*types.Project, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, nil
	}
	info, err := s.docker.InspectContainer(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("compose 快照不存在且读取运行态失败: %w", err)
	}
	project, err := compose.ProjectFromInspect(info)
	if err != nil {
		return nil, err
	}
	s.writeSnapshot(path, project, name)
	return project, nil
}

// GetComposeContent 读取容器对应的 compose 文件内容（YAML 文本）；
// 快照缺失时自动从运行态 inspect 反推并写入，然后返回内容。
func (s *SnapshotService) GetComposeContent(ctx context.Context, name string) (string, error) {
	path := s.path(name)
	if path == "" {
		return "", fmt.Errorf("容器名称或根目录未配置")
	}

	if _, err := s.ensureSnapshot(ctx, name, path); err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 compose 文件失败: %w", err)
	}
	return string(data), nil
}

// path 返回容器对应的 compose 快照完整路径；未配置根目录或名称为空时返回 ""
func (s *SnapshotService) path(name string) string {
	root := s.docker.ContainerRoot()
	if root == "" || name == "" {
		return ""
	}
	return filepath.Join(root, name, composeFileName)
}
