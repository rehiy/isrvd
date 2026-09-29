package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
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

func (b *commandOutputBuffer) String() string {
	return strings.ToValidUTF8(b.buf.String(), "?")
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

func (s *ExecSession) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *ExecSession) Write(p []byte) (int, error) { return s.writer.Write(p) }
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
	cmd := []string{shell, "-c", script}
	if shell == "" {
		cmd = []string{"/bin/sh", "-c", script}
	}

	execCfg := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          cmd,
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return "", err
	}

	attachResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", err
	}
	defer attachResp.Close()

	readCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		readCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}
	readDone := make(chan struct{})
	go func() {
		select {
		case <-readCtx.Done():
			attachResp.Close()
		case <-readDone:
		}
	}()

	buf := &commandOutputBuffer{}
	_, readErr := stdcopy.StdCopy(buf, buf, attachResp.Reader)
	close(readDone)
	output := buf.String()
	if buf.truncated {
		output += fmt.Sprintf("\n[output truncated at %d bytes]\n", maxCommandOutputBytes)
	}
	if err := readCtx.Err(); err != nil {
		return output, fmt.Errorf("容器命令执行中止: %w", err)
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return output, fmt.Errorf("读取容器命令输出失败: %w", readErr)
	}

	inspect, err := s.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return output, fmt.Errorf("检查容器命令状态失败: %w", err)
	}
	if inspect.Running {
		return output, fmt.Errorf("容器命令仍在运行")
	}
	if inspect.ExitCode != 0 {
		return output, fmt.Errorf("exit code %d", inspect.ExitCode)
	}
	return output, nil
}
