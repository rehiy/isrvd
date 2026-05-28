package docker

import (
	"context"
	"errors"
	"io"

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

	if info.Config != nil && info.Config.Tty {
		_, err = io.Copy(w, reader)
	} else {
		_, err = stdcopy.StdCopy(w, w, reader)
	}
	if err != nil && ctx.Err() == nil && !errors.Is(err, io.EOF) {
		logman.Warn("Container logs stream stopped with error", "id", req.ID, "error", err)
	}
}
