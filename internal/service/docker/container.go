package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/rehiy/libgo/logman"
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
	list, err := s.docker.ContainerList(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}
	return list, nil
}

// ContainerInspect 获取容器详情
func (s *Service) ContainerInspect(ctx context.Context, id string) (*pkgdocker.ContainerDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	detail, err := s.docker.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器详情失败: %w", err)
	}
	return detail, nil
}

// ContainerCreate 创建并启动容器
func (s *Service) ContainerCreate(ctx context.Context, req pkgdocker.ContainerSpec) (*ContainerCreateResult, error) {
	if req.Image == "" {
		return nil, fmt.Errorf("镜像名称不能为空")
	}
	if err := s.docker.ImageEnsure(ctx, req.Image, false); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", req.Image, err)
	}
	id, err := s.docker.ContainerCreate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	return &ContainerCreateResult{ID: pkgdocker.ShortID(id), Name: req.Name}, nil
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (*pkgdocker.ContainerStatsResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	stats, err := s.docker.ContainerStats(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器统计信息失败: %w", err)
	}
	return stats, nil
}

// ContainerAction 容器操作
func (s *Service) ContainerAction(ctx context.Context, req pkgdocker.ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("容器ID不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.ContainerAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("容器操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// ContainerLogs 获取容器日志
func (s *Service) ContainerLogs(ctx context.Context, req pkgdocker.ContainerLogsRequest) (*pkgdocker.ContainerLogsResult, error) {
	if req.ID == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	result, err := s.docker.ContainerLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取容器日志失败: %w", err)
	}
	return result, nil
}

// ContainerLogsStream 容器实时日志流
func (s *Service) ContainerLogsStream(ctx context.Context, w io.Writer, req pkgdocker.ContainerLogsRequest) {
	s.docker.ContainerLogsStream(ctx, w, req)
}

// ContainerExec 容器终端 WebSocket 桥接：建立 exec 会话后与 WebSocket 双向转发
func (s *Service) ContainerExec(ctx context.Context, conn *websocket.ServerConn, containerID, shell string) {
	session, err := s.docker.ContainerExecAttach(ctx, containerID, shell)
	if err != nil {
		conn.Write([]byte("[" + err.Error() + "]\r\n"))
		return
	}

	conn.Write([]byte("[容器终端已连接]\r\n"))

	// 转发容器输出到 WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := session.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					logman.Error("container exec read error", "error", err)
				}
				return
			}
		}
	}()

	// 转发 WebSocket 输入到容器
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logman.Error("websocket read error", "error", err)
			break
		}
		if n > 0 {
			if _, err := session.Write(buf[:n]); err != nil {
				logman.Error("container exec write error", "error", err)
				break
			}
		}
	}

	// 关闭 session 触发 reader goroutine 退出，等待完成后函数返回
	session.Close()
	<-done
}

// Info 获取 Docker 概览信息
func (s *Service) Info(ctx context.Context) (*pkgdocker.DockerInfo, error) {
	info, err := s.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Docker 信息失败: %w", err)
	}
	return info, nil
}
