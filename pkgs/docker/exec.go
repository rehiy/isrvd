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
	"github.com/rehiy/libgo/websocket"
)

// ContainerExec 容器终端 WebSocket 处理（业务逻辑层）
func (s *DockerService) ContainerExec(ctx context.Context, conn *websocket.ServerConn, containerID, shell string) {
	if shell == "" {
		shell = "/bin/sh"
	}

	// 创建 exec 实例
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		conn.Write([]byte("[创建终端会话失败: " + err.Error() + "]\r\n"))
		return
	}

	// 连接到 exec 实例
	attachConfig := container.ExecStartOptions{Tty: true}
	hijackedResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, attachConfig)
	if err != nil {
		conn.Write([]byte("[连接终端失败: " + err.Error() + "]\r\n"))
		return
	}
	defer hijackedResp.Close()

	conn.Write([]byte("[容器终端已连接]\r\n"))

	// 转发容器输出到 WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					logman.Error("Container exec read error", "error", err)
				}
				return
			}
			if n > 0 {
				conn.Write(buf[:n])
			}
		}
	}()

	// 转发 WebSocket 输入到容器
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logman.Error("WebSocket read error", "error", err)
			break
		}
		if n > 0 {
			if _, err := hijackedResp.Conn.Write(buf[:n]); err != nil {
				logman.Error("Container exec write error", "error", err)
				break
			}
		}
	}

	// 关闭 hijack 连接触发 reader goroutine 退出，等待其结束后函数返回
	hijackedResp.Close()
	<-done
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
func (s *DockerService) ContainerRunScript(ctx context.Context, image, shell, script string, timeout uint, volumes []VolumeMapping) (string, error) {
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
	for _, vol := range volumes {
		m, err := s.buildMount("", vol)
		if err != nil {
			return "", fmt.Errorf("处理挂载配置失败: %w", err)
		}
		mounts = append(mounts, m)
	}

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
