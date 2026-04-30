// Package docker 提供 Docker 业务服务层
package docker

import (
	"context"
	"fmt"

	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
	pkgdocker "isrvd/pkgs/docker"
)

// Service Docker 业务服务
type Service struct {
	docker *pkgdocker.DockerService
}

// NewService 创建 Docker 业务服务
func NewService() (*Service, error) {
	svc := registry.DockerService
	if svc == nil {
		logman.Error("Docker service not initialized")
		return nil, fmt.Errorf("Docker 服务未初始化")
	}
	return &Service{docker: svc}, nil
}

// CheckAvailability 检测 Docker 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.docker == nil {
		return false
	}
	_, err := s.docker.GetInfo(ctx)
	return err == nil
}

// GetDockerService 返回底层 pkgs/docker.DockerService（供 WebSocket exec 使用）
func (s *Service) GetDockerService() *pkgdocker.DockerService {
	return s.docker
}
