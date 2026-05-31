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

// ContainerLogsRequest 日志请求
type ContainerLogsRequest struct {
	ID   string `json:"id" binding:"required"`
	Tail string `json:"tail"`
}

// ContainerLogsResult 容器日志结果
type ContainerLogsResult struct {
	ID   string   `json:"id"`
	Logs []string `json:"logs"`
}

// ContainerLogs 获取容器日志快照。
func (s *DockerService) ContainerLogs(ctx context.Context, req ContainerLogsRequest) (*ContainerLogsResult, error) {
	if req.Tail == "" {
		req.Tail = "100"
	}

	// 获取容器信息以判断是否为 TTY 模式
	info, err := s.client.ContainerInspect(ctx, req.ID)
	if err != nil {
		logman.Error("Inspect container for logs failed", "id", req.ID, "error", err)
		return nil, err
	}

	reader, err := s.client.ContainerLogs(ctx, req.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       req.Tail,
		Follow:     false,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Get container logs failed", "id", req.ID, "error", err)
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logman.Error("Read container logs failed", "id", req.ID, "error", err)
		return nil, err
	}

	// TTY 模式下日志不带 8 字节帧头，直接作为纯文本处理
	if info.Config != nil && info.Config.Tty {
		return &ContainerLogsResult{ID: req.ID, Logs: []string{string(data)}}, nil
	}
	return &ContainerLogsResult{ID: req.ID, Logs: ParseDockerLogs(data)}, nil
}

// ContainerLogsStream 实时转发容器日志到 writer。
// writer 可选实现 sse.Writer 以区分 error 事件与普通 data 事件。
func (s *DockerService) ContainerLogsStream(ctx context.Context, w io.Writer, req ContainerLogsRequest) {
	if req.Tail == "" {
		req.Tail = "100"
	}

	writeError := func(msg string) {
		if sw, ok := w.(httpd.Writer); ok {
			_ = sw.WriteEvent("error", msg)
		} else {
			_, _ = w.Write([]byte("[" + msg + "]\n"))
		}
	}

	info, err := s.client.ContainerInspect(ctx, req.ID)
	if err != nil {
		logman.Error("Inspect container for logs stream failed", "id", req.ID, "error", err)
		writeError("获取容器信息失败: " + err.Error())
		return
	}

	reader, err := s.client.ContainerLogs(ctx, req.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       req.Tail,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Start container logs stream failed", "id", req.ID, "error", err)
		writeError("获取容器日志失败: " + err.Error())
		return
	}
	defer reader.Close()

	// 心跳 ticker，保持 SSE 连接活跃
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	// 使用带上下文的复制，检测连接断开
	errCh := make(chan error, 1)
	go func() {
		var copyErr error
		if info.Config != nil && info.Config.Tty {
			_, copyErr = io.Copy(w, reader)
		} else {
			_, copyErr = stdcopy.StdCopy(w, w, reader)
		}
		errCh <- copyErr
	}()

	for {
		select {
		case err := <-errCh:
			// 复制完成或出错
			if err != nil && ctx.Err() == nil && !errors.Is(err, io.EOF) {
				logman.Warn("Container logs stream stopped with error", "id", req.ID, "error", err)
			}
			return
		case <-ctx.Done():
			// 上下文取消（客户端断开）
			// reader.Close() 会被 defer 调用，io.Copy 会很快返回
			logman.Info("Container logs stream cancelled by context", "id", req.ID)
			return
		case <-heartbeat.C:
			// 发送心跳保持连接
			if sw, ok := w.(httpd.Writer); ok {
				if err := sw.WriteEvent("heartbeat", "ping"); err != nil {
					logman.Warn("Failed to send heartbeat", "id", req.ID, "error", err)
				}
			}
		}
	}
}
