package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"
)

const maxLogSnapshotBytes int64 = 4 << 20

var errLogSnapshotFull = errors.New("日志快照已达到上限")

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

	logs, err := ReadLogSnapshot(reader, info.Config != nil && info.Config.Tty)
	if err != nil {
		logman.Error("Read container logs failed", "id", id, "error", err)
		return nil, err
	}
	return logs, nil
}

type logSnapshotWriter struct {
	logs      []string
	remaining int64
	truncated bool
}

func (w *logSnapshotWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if w.remaining <= 0 {
		w.truncated = true
		return 0, errLogSnapshotFull
	}
	if int64(len(p)) > w.remaining {
		n := int(w.remaining)
		w.logs = append(w.logs, string(p[:n]))
		w.remaining = 0
		w.truncated = true
		return n, errLogSnapshotFull
	}
	w.logs = append(w.logs, string(p))
	w.remaining -= int64(len(p))
	return len(p), nil
}

// ReadLogSnapshot 读取默认大小限制的 Docker 日志快照，供容器与 Swarm 共用。
func ReadLogSnapshot(reader io.Reader, tty bool) ([]string, error) {
	return readLogSnapshot(reader, tty, maxLogSnapshotBytes)
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

// ─── 辅助函数 ───

func readLogSnapshot(reader io.Reader, tty bool, limit int64) ([]string, error) {
	if limit < 0 {
		return nil, fmt.Errorf("日志读取上限不能为负数")
	}
	writer := &logSnapshotWriter{remaining: limit}
	var err error
	if tty {
		_, err = io.Copy(writer, reader)
	} else {
		_, err = stdcopy.StdCopy(writer, writer, reader)
	}
	truncated := errors.Is(err, errLogSnapshotFull) || writer.truncated
	if err != nil && !truncated {
		return nil, err
	}
	logs := writer.logs
	if tty && len(logs) > 1 {
		logs = []string{strings.Join(logs, "")}
	}
	if truncated {
		logs = append(logs, fmt.Sprintf("\n[output truncated at %d bytes]\n", limit))
	}
	return logs, nil
}
