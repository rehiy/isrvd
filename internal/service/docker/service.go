// Package docker 提供 Docker 业务服务层
package docker

import (
	"context"
	"fmt"

	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
	pkgdocker "isrvd/pkgs/docker"
)

// SnapshotSaver 快照保存接口，解耦对 compose service 的直接依赖
type SnapshotSaver interface {
	Save(req pkgdocker.ContainerCreateRequest)
}

// Service Docker 业务服务
type Service struct {
	docker   *pkgdocker.DockerService
	snapshot SnapshotSaver // 可为 nil，创建/更新容器时安静跳过
}

// NewService 创建 Docker 业务服务
func NewService(snapshot SnapshotSaver) (*Service, error) {
	svc := registry.DockerService
	if svc == nil {
		logman.Error("Docker service not initialized")
		return nil, fmt.Errorf("Docker 服务未初始化")
	}
	return &Service{docker: svc, snapshot: snapshot}, nil
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

// saveSnapshot 安静地保存快照（失败仅记日志）
func (s *Service) saveSnapshot(req pkgdocker.ContainerCreateRequest) {
	if s.snapshot != nil {
		s.snapshot.Save(req)
	}
}
