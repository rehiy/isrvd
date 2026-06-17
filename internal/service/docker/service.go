// Package docker 提供 Docker 业务服务层
package docker

import (
	"context"
	"fmt"

	"github.com/rehiy/libgo/logman"

	"isrvd/internal/registry"
	pkgDocker "isrvd/pkgs/docker"
)

// Service Docker 业务服务
type Service struct {
	docker *pkgDocker.DockerService
}

// NewService 创建 Docker 业务服务
func NewService() (*Service, error) {
	svc := registry.DockerService
	if svc == nil {
		logman.Warn("Docker service not initialized")
		return nil, fmt.Errorf("Docker 服务未初始化")
	}
	return &Service{docker: svc}, nil
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

// ActionRequest 资源操作请求（容器/镜像/网络/卷通用）。
type ActionRequest struct {
	ID     string `json:"id" binding:"required"`     // 目标资源 ID
	Action string `json:"action" binding:"required"` // 操作动作（如 start/stop/restart/remove）
}
