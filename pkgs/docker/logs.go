package docker

import (
	"context"
	"errors"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
)

// ContainerLogsStream 实时转发容器日志到 WebSocket。
func (s *DockerService) ContainerLogsStream(ctx context.Context, conn *websocket.ServerConn, id, tail string) {
	defer conn.Close()

	if tail == "" {
		tail = "100"
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := conn.Read(buf); err != nil {
				cancel()
				return
			}
		}
	}()

	info, err := s.client.ContainerInspect(streamCtx, id)
	if err != nil {
		logman.Error("Inspect container for logs stream failed", "id", id, "error", err)
		_, _ = conn.Write([]byte("[获取容器信息失败: " + err.Error() + "]\n"))
		return
	}

	reader, err := s.client.ContainerLogs(streamCtx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Start container logs stream failed", "id", id, "error", err)
		_, _ = conn.Write([]byte("[获取容器日志失败: " + err.Error() + "]\n"))
		return
	}
	defer reader.Close()

	if info.Config != nil && info.Config.Tty {
		_, err = io.Copy(conn, reader)
	} else {
		_, err = stdcopy.StdCopy(conn, conn, reader)
	}
	if err != nil && streamCtx.Err() == nil && !errors.Is(err, io.EOF) {
		logman.Warn("Container logs stream stopped with error", "id", id, "error", err)
	}
}
