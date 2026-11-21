package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const maxCommandOutputBytes int64 = 4 << 20

type commandOutputBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (b *commandOutputBuffer) Write(p []byte) (int, error) {
	remaining := maxCommandOutputBytes - int64(b.buf.Len())
	if remaining > 0 {
		keep := min(int64(len(p)), remaining)
		_, _ = b.buf.Write(p[:int(keep)])
	}
	if int64(len(p)) > remaining {
		b.truncated = true
	}
	return len(p), nil
}

// ExecSession 容器 exec 会话，封装 hijacked 连接，实现 io.ReadWriteCloser
type ExecSession struct {
	client *DockerService
	ctx    context.Context
	execID string
	reader io.Reader
	writer io.Writer
	closer func() // hijackedResp.Close() 返回 void，用 func() 封装
}

func (s *ExecSession) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}
func (s *ExecSession) Write(p []byte) (int, error) {
	return s.writer.Write(p)
}
func (s *ExecSession) Close() error {
	s.closer()
	return nil
}

func (s *ExecSession) Resize(cols, rows int) error {
	if s.client == nil || s.execID == "" {
		return nil
	}
	return s.client.client.ContainerExecResize(s.ctx, s.execID, container.ResizeOptions{
		Width:  uint(cols),
		Height: uint(rows),
	})
}

// ContainerExecAttach 创建并连接容器 exec 会话。
// 调用方负责关闭返回的 session。
func (s *DockerService) ContainerExecAttach(ctx context.Context, containerID, shell string) (*ExecSession, error) {
	if shell == "" {
		shell = "/bin/sh"
	}

	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("创建终端会话失败: %w", err)
	}

	hijackedResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{Tty: true})
	if err != nil {
		return nil, fmt.Errorf("连接终端失败: %w", err)
	}

	// closer 调用 hijackedResp.Close()，它会同时关闭底层连接的读端和写端，
	// 避免只关闭 Conn 导致 reader goroutine 永久阻塞在 Read 上
	return &ExecSession{
		client: s,
		ctx:    ctx,
		execID: execResp.ID,
		reader: hijackedResp.Reader,
		writer: hijackedResp.Conn,
		closer: hijackedResp.Close,
	}, nil
}

// ContainerExecRun 在指定容器内非交互地执行命令，返回合并后的 stdout/stderr 输出。
// timeout 为 0 时不限制。
func (s *DockerService) ContainerExecRun(ctx context.Context, containerID, shell, script string, timeout uint) (string, error) {
	output, truncated, err := s.containerExecRun(ctx, containerID, shell, script, timeout)
	output = strings.ToValidUTF8(output, "?")
	if truncated {
		output += fmt.Sprintf("\n[output truncated at %d bytes]\n", maxCommandOutputBytes)
	}
	return output, err
}

func (s *DockerService) containerExecRun(ctx context.Context, containerID, shell, script string, timeout uint) (string, bool, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}
	if shell == "" {
		shell = "/bin/sh"
	}
	execResp, err := s.client.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{shell, "-c", script},
	})
	if err != nil {
		return "", false, err
	}
	body, err := s.execOutput(ctx, execResp.ID)
	if err != nil {
		return "", false, err
	}
	defer body.Close()
	stop := context.AfterFunc(ctx, func() { _ = body.Close() })
	defer stop()

	buf := &commandOutputBuffer{}
	_, readErr := stdcopy.StdCopy(buf, buf, body)
	output := buf.buf.String()
	if err := ctx.Err(); err != nil {
		return output, buf.truncated, fmt.Errorf("容器命令执行中止: %w", err)
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return output, buf.truncated, fmt.Errorf("读取容器命令输出失败: %w", readErr)
	}
	inspect, err := s.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return output, buf.truncated, fmt.Errorf("检查容器命令状态失败: %w", err)
	}
	if inspect.Running {
		return output, buf.truncated, fmt.Errorf("容器命令仍在运行")
	}
	if inspect.ExitCode != 0 {
		return output, buf.truncated, commandExitError(inspect.ExitCode)
	}
	return output, buf.truncated, nil
}

type commandExitError int

func (e commandExitError) Error() string {
	return fmt.Sprintf("exit code %d", e)
}

// execOutput 使用 SDK 的 HTTP 传输读取升级响应，使等待响应头也受请求上下文约束。
func (s *DockerService) execOutput(ctx context.Context, id string) (io.ReadCloser, error) {
	host, err := client.ParseHostURL(s.client.DaemonHost())
	if err != nil {
		return nil, err
	}
	httpClient := s.client.HTTPClient()
	address := host.Host
	if host.Scheme == "unix" || host.Scheme == "npipe" {
		address = client.DummyHost
	}
	version := "v" + strings.TrimPrefix(s.client.ClientVersion(), "v")
	if version == "v" {
		version = ""
	}
	target := url.URL{Scheme: s.execScheme, Host: address, Path: path.Join(host.Path, version, "exec", id, "start")}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		resp.Body.Close()
		return nil, fmt.Errorf("连接容器执行输出失败: HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}
