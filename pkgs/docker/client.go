package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"
)

// DockerService Docker 服务
type DockerService struct {
	client *client.Client
	config *DockerConfig
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

// GetClient 获取 Docker 客户端
func (s *DockerService) GetClient() *client.Client {
	return s.client
}

// ContainerRoot 获取容器数据根目录
func (s *DockerService) ContainerRoot() string {
	if s.config == nil {
		return ""
	}
	return s.config.ContainerRoot
}

// DockerInfo Docker 信息概览
type DockerInfo struct {
	ContainersRunning  int64    `json:"containersRunning"`
	ContainersStopped  int64    `json:"containersStopped"`
	ContainersPaused   int64    `json:"containersPaused"`
	ImagesTotal        int64    `json:"imagesTotal"`
	VolumesTotal       int64    `json:"volumesTotal"`
	NetworksTotal      int64    `json:"networksTotal"`
	RegistryMirrors    []string `json:"registryMirrors"`
	IndexServerAddress string   `json:"indexServerAddress"`
}

// GetInfo 获取 Docker 概览信息
func (s *DockerService) GetInfo(ctx context.Context) (*DockerInfo, error) {
	daemonInfo, err := s.client.Info(ctx)
	if err != nil {
		logman.Error("Docker info failed", "error", err)
		return nil, err
	}

	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logman.Error("Container list failed", "error", err)
		return nil, err
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

	images, _ := s.client.ImageList(ctx, image.ListOptions{All: true})
	volList, _ := s.client.VolumeList(ctx, volume.ListOptions{})
	networks, _ := s.client.NetworkList(ctx, network.ListOptions{})

	// 读取镜像加速器配置
	var mirrors []string
	if daemonInfo.RegistryConfig != nil {
		mirrors = daemonInfo.RegistryConfig.Mirrors
	}

	return &DockerInfo{
		ContainersRunning:  running,
		ContainersStopped:  stopped,
		ContainersPaused:   paused,
		ImagesTotal:        int64(len(images)),
		VolumesTotal:       int64(len(volList.Volumes)),
		NetworksTotal:      int64(len(networks)),
		RegistryMirrors:    mirrors,
		IndexServerAddress: daemonInfo.IndexServerAddress,
	}, nil
}
