package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"
)

// DockerService Docker 服务
type DockerService struct {
	client *client.Client
	config *DockerConfig
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

// GetInfo 获取 Docker 概览信息
func (s *DockerService) GetInfo(ctx context.Context) (*DockerInfo, error) {
	_, err := s.client.Info(ctx)
	if err != nil {
		logman.Error("Docker info failed", "error", err)
		return nil, err
	}

	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{All: true})
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

	images, _ := s.client.ImageList(ctx, types.ImageListOptions{All: true})
	volList, _ := s.client.VolumeList(ctx, volume.ListOptions{})
	networks, _ := s.client.NetworkList(ctx, types.NetworkListOptions{})

	return &DockerInfo{
		ContainersRunning: running,
		ContainersStopped: stopped,
		ContainersPaused:  paused,
		ImagesTotal:       int64(len(images)),
		VolumesTotal:      int64(len(volList.Volumes)),
		NetworksTotal:     int64(len(networks)),
	}, nil
}
