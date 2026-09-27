package docker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"
)

// ContainerLogs 获取容器日志快照。
func (s *DockerService) ContainerLogs(ctx context.Context, id, tail string) ([]string, error) {
	if tail == "" {
		tail = "100"
	}

	// 获取容器信息以判断是否为 TTY 模式
	info, err := s.client.ContainerInspect(ctx, id)
	if err != nil {
		logman.Error("Inspect container for logs failed", "id", id, "error", err)
		return nil, err
	}

	reader, err := s.client.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     false,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Get container logs failed", "id", id, "error", err)
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logman.Error("Read container logs failed", "id", id, "error", err)
		return nil, err
	}

	// TTY 模式下日志不带 8 字节帧头，直接作为纯文本处理
	if info.Config != nil && info.Config.Tty {
		return []string{string(data)}, nil
	}
	return ParseDockerLogs(data), nil
}

// ContainerLogsStream 实时转发容器日志到 writer。
// writer 可选实现 httpd.Writer 以区分 error 事件与普通 data 事件。
func (s *DockerService) ContainerLogsStream(ctx context.Context, w io.Writer, id, tail string) {
	if tail == "" {
		tail = "100"
	}

	info, err := s.client.ContainerInspect(ctx, id)
	if err != nil {
		logman.Error("Inspect container for logs stream failed", "id", id, "error", err)
		LogErrorWrite(w, "获取容器信息失败: "+err.Error())
		return
	}

	reader, err := s.client.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Start container logs stream failed", "id", id, "error", err)
		LogErrorWrite(w, "获取容器日志失败: "+err.Error())
		return
	}
	cancelled, err := LogStream(ctx, w, reader, info.Config != nil && info.Config.Tty, id)
	if err != nil {
		logman.Warn("Container logs stream stopped with error", "id", id, "error", err)
	} else if cancelled {
		logman.Info("Container logs stream cancelled by context", "id", id)
	}
}

// ─── 辅助函数 ───

// LogStream 转发 Docker 日志流、发送 SSE 心跳，并在上下文取消时关闭 reader。
// tty 为 true 时直接复制文本，否则使用 Docker 多路复用帧格式解码。
// 返回值分别表示是否因上下文取消而停止、复制过程中发生的错误。
func LogStream(ctx context.Context, w io.Writer, reader io.ReadCloser, tty bool, id string) (bool, error) {
	defer reader.Close()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	errCh := make(chan error, 1)
	go func() {
		var copyErr error
		if tty {
			_, copyErr = io.Copy(w, reader)
		} else {
			_, copyErr = stdcopy.StdCopy(w, w, reader)
		}
		errCh <- copyErr
	}()
	for {
		select {
		case err := <-errCh:
			if err != nil && ctx.Err() == nil && !errors.Is(err, io.EOF) {
				return false, err
			}
			return false, nil
		case <-ctx.Done():
			return true, nil
		case <-heartbeat.C:
			if sw, ok := w.(httpd.Writer); ok {
				if err := sw.WriteEvent("heartbeat", "ping"); err != nil {
					logman.Warn("Failed to send heartbeat", "id", id, "error", err)
				}
			}
		}
	}
}

// LogErrorWrite 将日志读取错误写为 SSE error 事件，普通 writer 使用文本格式。
func LogErrorWrite(w io.Writer, message string) {
	if sw, ok := w.(httpd.Writer); ok {
		_ = sw.WriteEvent("error", message)
	} else {
		_, _ = w.Write([]byte("[" + message + "]\n"))
	}
}
