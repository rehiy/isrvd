package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/rehiy/libgo/websocket"

	pkgdocker "isrvd/pkgs/docker"
)

// ContainerCreateResult 创建容器结果
type ContainerCreateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContainerList 列出容器
func (s *Service) ContainerList(ctx context.Context, all bool) ([]*pkgdocker.ContainerInfo, error) {
	return s.docker.ContainerList(ctx, all)
}

// ContainerInspect 获取容器详情
func (s *Service) ContainerInspect(ctx context.Context, id string) (*pkgdocker.ContainerDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	return s.docker.ContainerInspect(ctx, id)
}

// ContainerCreate 创建并启动容器
func (s *Service) ContainerCreate(ctx context.Context, req pkgdocker.ContainerSpec) (*ContainerCreateResult, error) {
	if err := s.docker.ImageEnsure(ctx, req.Image, false); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", req.Image, err)
	}
	id, err := s.docker.ContainerCreate(ctx, req)
	if err != nil {
		return nil, err
	}
	return &ContainerCreateResult{ID: pkgdocker.ShortID(id), Name: req.Name}, nil
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (*pkgdocker.ContainerStatsResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	return s.docker.ContainerStats(ctx, id)
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req pkgdocker.ActionRequest) error {
	return s.docker.ContainerAction(ctx, req.ID, req.Action)
}

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req pkgdocker.ContainerLogsRequest) (*pkgdocker.ContainerLogsResult, error) {
	return s.docker.ContainerLogs(ctx, req)
}

// ContainerLogsStream 容器实时日志流（代理到 pkgs 层）
func (s *Service) ContainerLogsStream(ctx context.Context, w io.Writer, req pkgdocker.ContainerLogsRequest) {
	s.docker.ContainerLogsStream(ctx, w, req)
}

// ContainerExec 容器终端 WebSocket（代理到 pkgs 层）
func (s *Service) ContainerExec(ctx context.Context, conn *websocket.ServerConn, containerID, shell string) {
	s.docker.ContainerExec(ctx, conn, containerID, shell)
}

// Info 获取 Docker 概览信息
func (s *Service) Info(ctx context.Context) (*pkgdocker.DockerInfo, error) {
	return s.docker.Info(ctx)
}
