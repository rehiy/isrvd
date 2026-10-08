package docker

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
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

	remote bool // daemon 是否位于 isrvd 之外的主机，见 isRemoteHost

	selfID     string
	selfIDOnce sync.Once
	closeOnce  sync.Once
	closeErr   error
}

// DockerConfig Docker 配置（由外部注入，解除对 config 的依赖）
type DockerConfig struct {
	Host          string            // Docker 连接地址
	TLS           *TLSConfig        // 远程 daemon 的 TLS 参数；为 nil 表示不使用 TLS
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
	if cfg.TLS != nil {
		tlsConfig, err := cfg.TLS.Config(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("Docker TLS 配置无效: %w", err)
		}
		opts = append(opts, withTLSConfig(tlsConfig))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		logman.Error("Docker client init failed", "error", err)
		return nil, err
	}

	return &DockerService{client: cli, config: cfg, remote: isRemoteHost(cfg.Host)}, nil
}

// Client 获取 Docker 客户端
func (s *DockerService) Client() *client.Client {
	return s.client
}

// Close 幂等关闭底层 Docker SDK 客户端。
func (s *DockerService) Close() error {
	s.closeOnce.Do(func() {
		if s.client != nil {
			s.closeErr = s.client.Close()
		}
	})
	return s.closeErr
}

// Remote 返回 daemon 是否可能位于 isrvd 之外的主机（tcp:// 且不是回环地址）。
// 这类 daemon 上的容器不能靠本机的 IP、主机名识别自身（两台主机常常都使用 172.17.0.x），
// 也不能用本机文件系统检查 bind 挂载源。判定只看地址，与是否启用 TLS 无关。
func (s *DockerService) Remote() bool {
	return s != nil && s.remote
}

// isRemoteHost 判断 Docker Host 是否指向 isrvd 之外的主机。
// host 为空时与 SDK 一致，回退到环境变量 DOCKER_HOST。
// unix://、npipe:// 与回环地址的 tcp:// 视为本机；其余 tcp:// 视为远程，
// 包括 tcp://docker-proxy:2375 这类容器网络内的别名：无法确认它与 isrvd 同机，按远程保守处理。
func isRemoteHost(host string) bool {
	if strings.TrimSpace(host) == "" {
		host = os.Getenv(client.EnvOverrideHost)
	}
	proto, addr, ok := strings.Cut(strings.TrimSpace(host), "://")
	if !ok || proto != "tcp" {
		return false
	}
	name := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		name = h
	}
	name = strings.Trim(name, "[]")
	if strings.EqualFold(name, "localhost") {
		return false
	}
	if ip := net.ParseIP(name); ip != nil && ip.IsLoopback() {
		return false
	}
	return true
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
	registries := make([]*RegistryConfig, len(s.config.Registries))
	for i, registry := range s.config.Registries {
		registries[i] = cloneRegistry(registry)
	}
	return registries
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
