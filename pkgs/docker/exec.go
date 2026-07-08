package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/rehiy/libgo/logman"
)

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

	// 使用 context 控制读取超时：cancel 后关闭连接使 io.Copy 中断
	if timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
		go func() {
			<-timeoutCtx.Done()
			attachResp.Close()
		}()
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, attachResp.Reader); err != nil && err != io.EOF {
		logman.Warn("ContainerExecRun read error", "container", containerID, "error", err)
	}

	inspect, err := s.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return buf.String(), nil
	}
	if inspect.ExitCode != 0 {
		return buf.String(), fmt.Errorf("exit code %d", inspect.ExitCode)
	}
	return buf.String(), nil
}

// ContainerRunScript 创建临时容器运行脚本，完成后收集日志并删除容器和临时文件。
// 脚本内容通过临时文件 bind mount 进容器执行，不依赖现有容器。
// image: 镜像名；shell: 容器内 shell（默认 /bin/sh）；script: 脚本内容；timeout: 超时秒数（0 不限）。
func (s *DockerService) ContainerRunScript(ctx context.Context, image, shell, script string, timeout uint, extraMounts []mount.Mount) (string, error) {
	if err := s.ImageEnsure(ctx, image, false); err != nil {
		return "", fmt.Errorf("镜像 %s 不可用: %w", image, err)
	}

	// 写脚本到宿主机临时文件
	tmpFile, err := os.CreateTemp("", "cron-script-*.sh")
	if err != nil {
		return "", fmt.Errorf("创建临时脚本文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("写入临时脚本文件失败: %w", err)
	}
	tmpFile.Close()
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return "", fmt.Errorf("设置脚本权限失败: %w", err)
	}

	if shell == "" {
		shell = "/bin/sh"
	}
	scriptInContainer := "/tmp/" + filepath.Base(tmpPath)

	// 构建 hostConfig
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   tmpPath,
			Target:   scriptInContainer,
			ReadOnly: true,
		},
	}
	mounts = append(mounts, extraMounts...)

	containerName := fmt.Sprintf("cron-%x", time.Now().UnixNano())
	containerCfg := &container.Config{
		Image: image,
		Cmd:   []string{shell, scriptInContainer},
	}
	hostCfg := &container.HostConfig{
		Mounts:      mounts,
		NetworkMode: "none",
		AutoRemove:  false, // 手动删除以便读日志
	}

	resp, err := s.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("创建临时容器失败: %w", err)
	}
	defer s.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := s.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("启动临时容器失败: %w", err)
	}

	// 等待容器退出
	waitCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	statusCh, errCh := s.client.ContainerWait(waitCtx, resp.ID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitResult := <-statusCh:
		exitCode = waitResult.StatusCode
		if waitResult.Error != nil {
			logman.Warn("ContainerRunScript wait error", "container", containerName, "error", waitResult.Error.Message)
		}
	case err := <-errCh:
		_ = s.client.ContainerStop(ctx, resp.ID, container.StopOptions{})
		return "", fmt.Errorf("等待容器退出失败: %w", err)
	}

	// 读取日志
	logReader, err := s.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	output := ""
	if err == nil {
		defer logReader.Close()
		raw, _ := io.ReadAll(logReader)
		output = strings.Join(ParseDockerLogs(raw), "")
	}

	if exitCode != 0 {
		return output, fmt.Errorf("脚本退出码 %d", exitCode)
	}
	return output, nil
}
