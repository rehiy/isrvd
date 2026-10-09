// Package docker 提供 Docker 业务服务层
package docker

import (
	"context"
	"fmt"

	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/docker"
	"isrvd/server/config"
)

// Service Docker 业务服务
type Service struct {
	docker *docker.DockerService
}

// NewService 创建 Docker 业务服务
func NewService() (*Service, error) {
	snapshot := config.Current()
	var registries []*docker.RegistryConfig
	for _, reg := range snapshot.Docker.Registries {
		if reg == nil {
			continue
		}
		registries = append(registries, &docker.RegistryConfig{
			Name:        reg.Name,
			Description: reg.Description,
			URL:         reg.URL,
			Username:    reg.Username,
			Password:    reg.Password,
		})
	}

	var tlsConfig *docker.TLSConfig
	if t := snapshot.Docker.TLS; t != nil && t.Enabled {
		tlsConfig = &docker.TLSConfig{SkipVerify: t.SkipVerify, CA: t.CA, Cert: t.Cert, Key: t.Key}
	}

	svc, err := docker.NewDockerService(&docker.DockerConfig{
		Host:          snapshot.Docker.Host,
		TLS:           tlsConfig,
		ContainerRoot: snapshot.Docker.ContainerRoot,
		Registries:    registries,

		ContainerGuard: selfContainerGuard,
	})
	if err != nil {
		return nil, fmt.Errorf("Docker 服务初始化失败: %w", err)
	}
	return &Service{docker: svc}, nil
}

// selfContainerGuard 禁止对 iSrvd 自身所在容器执行会中断自身的操作。
// 通过 DockerConfig.ContainerGuard 注入，所有复用同一 DockerService 的服务（compose 等）一并生效。
func selfContainerGuard(ctx context.Context, s *docker.DockerService, id, action string) error {
	switch action {
	case "stop", "restart", "remove", "pause":
		selfID := s.SelfContainerID(ctx)
		if selfID != "" && (id == selfID || docker.ShortID(id) == docker.ShortID(selfID)) {
			return fmt.Errorf("禁止操作当前 iSrvd 所在容器")
		}
	}
	return nil
}

// Raw 返回底层 Docker 客户端，供 swarm/cron/monitor/compose 等依赖 Docker 的服务复用。
// s 为 nil（Docker 不可用）时安全返回 nil。
func (s *Service) Raw() *docker.DockerService {
	if s == nil {
		return nil
	}
	return s.docker
}

// CheckAvailability 检测 Docker 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.docker == nil {
		return false
	}
	_, err := s.docker.Info(ctx)
	return err == nil
}

// DockerInfo Docker 信息概览，保持前端稳定响应结构。
type DockerInfo struct {
	ContainersRunning  int64    `json:"containersRunning"`  // 运行中的容器数
	ContainersStopped  int64    `json:"containersStopped"`  // 已停止的容器数
	ContainersPaused   int64    `json:"containersPaused"`   // 已暂停的容器数
	ImagesTotal        int64    `json:"imagesTotal"`        // 镜像总数
	VolumesTotal       int64    `json:"volumesTotal"`       // 卷总数
	NetworksTotal      int64    `json:"networksTotal"`      // 网络总数
	RegistryMirrors    []string `json:"registryMirrors"`    // 镜像加速器地址列表
	IndexServerAddress string   `json:"indexServerAddress"` // 默认镜像仓库地址
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

// ActionRequest 资源操作请求（容器/镜像/网络/卷通用）。
type ActionRequest struct {
	ID     string `json:"id" binding:"required"`     // 目标资源 ID
	Action string `json:"action" binding:"required"` // 操作动作（如 start/stop/restart/remove）
}
