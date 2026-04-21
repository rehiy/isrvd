package docker

import (
	"context"
	"fmt"

	pkgdocker "isrvd/pkgs/docker"
)

// ContainerCreateResult 创建容器结果
type ContainerCreateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContainerLogsResult 容器日志结果
type ContainerLogsResult struct {
	ID   string   `json:"id"`
	Logs []string `json:"logs"`
}

// ListContainers 列出容器
func (s *Service) ListContainers(ctx context.Context, all bool) (any, error) {
	return s.docker.ListContainers(ctx, all)
}

// CreateContainer 创建容器
func (s *Service) CreateContainer(ctx context.Context, req pkgdocker.ContainerCreateRequest) (*ContainerCreateResult, error) {
	id, err := s.docker.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	shortID := id
	if len(id) > 12 {
		shortID = id[:12]
	}
	return &ContainerCreateResult{ID: shortID, Name: req.Name}, nil
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	return s.docker.GetContainerStats(ctx, id)
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req pkgdocker.ContainerActionRequest) error {
	return s.docker.ContainerAction(ctx, req.ID, req.Action)
}

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req pkgdocker.ContainerLogsRequest) (*ContainerLogsResult, error) {
	logs, err := s.docker.GetContainerLogs(ctx, req.ID, req.Tail)
	if err != nil {
		return nil, err
	}
	return &ContainerLogsResult{ID: req.ID, Logs: logs}, nil
}

// Info 获取 Docker 概览信息
func (s *Service) Info(ctx context.Context) (any, error) {
	return s.docker.GetInfo(ctx)
}
