package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/rehiy/libgo/logman"
)

// DockerService Docker 服务
type DockerService struct {
	client     *client.Client
	config     *DockerConfig
	registryMu sync.RWMutex // 保护 config.Registries 的并发读写

	selfID     string
	selfIDOnce sync.Once
}

// DockerConfig Docker 配置（由外部注入，解除对 config 的依赖）
type DockerConfig struct {
	Host          string            // Docker 连接地址
	ContainerRoot string            // 容器数据根目录
	Registries    []*RegistryConfig // 镜像仓库配置列表
}

// RegistryConfig 镜像仓库配置
type RegistryConfig struct {
	Name        string // 仓库名称
	URL         string // 仓库地址
	Username    string // 用户名
	Password    string // 密码
	Description string // 仓库描述
}

// NewDockerService 创建 Docker 服务
func NewDockerService(cfg *DockerConfig) (*DockerService, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	} else {
		opts = append(opts, client.FromEnv)
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		logman.Error("Docker client init failed", "error", err)
		return nil, err
	}

	return &DockerService{client: cli, config: cfg}, nil
}

// Client 获取 Docker 客户端
func (s *DockerService) Client() *client.Client {
	return s.client
}

// ContainerRoot 获取容器数据根目录
func (s *DockerService) ContainerRoot() string {
	if s.config == nil {
		return ""
	}
	return s.config.ContainerRoot
}

// Registries 返回全量仓库配置（包含密码），仅供上层持久化使用
func (s *DockerService) Registries() []*RegistryConfig {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	return s.config.Registries
}

// Info 获取 Docker daemon 原始信息。
func (s *DockerService) Info(ctx context.Context) (system.Info, error) {
	info, err := s.client.Info(ctx)
	if err != nil {
		logman.Error("Docker info failed", "error", err)
		return system.Info{}, err
	}
	return info, nil
}
