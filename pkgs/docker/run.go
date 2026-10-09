package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/rehiy/libgo/logman"
)

// ScriptRunOptions 临时容器运行脚本的参数。
type ScriptRunOptions struct {
	Image      string        // 镜像名
	Shell      string        // 容器内 shell，默认 /bin/sh
	Script     string        // 脚本内容
	Timeout    uint          // 超时秒数，0 不限
	Mounts     []mount.Mount // 额外挂载
	NamePrefix string        // 临时容器名前缀，默认 script
	ScriptName string        // 脚本在容器根目录下的文件名，默认 script.sh
}

// ContainerRunScript 创建临时容器运行脚本，完成后收集日志并删除容器。
// 脚本内容通过 Docker Copy API 写入容器，兼容远程 daemon 和容器化部署。
func (s *DockerService) ContainerRunScript(ctx context.Context, opts ScriptRunOptions) (string, error) {
	image, script, timeout := opts.Image, opts.Script, opts.Timeout
	if err := s.ImageEnsure(ctx, image, false); err != nil {
		return "", fmt.Errorf("镜像 %s 不可用: %w", image, err)
	}

	shell := opts.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	namePrefix := opts.NamePrefix
	if namePrefix == "" {
		namePrefix = "script"
	}
	scriptName := opts.ScriptName
	if scriptName == "" {
		scriptName = "script.sh"
	}
	const scriptDir = "/"
	scriptInContainer := scriptDir + scriptName

	// 仅保留调用方指定的挂载；脚本通过 Docker Copy API 写入，兼容容器化部署和远程 daemon。
	mounts := append([]mount.Mount(nil), opts.Mounts...)

	containerName := fmt.Sprintf("%s-%x", namePrefix, time.Now().UnixNano())
	containerCfg := &container.Config{
		Image:      image,
		Entrypoint: []string{shell},
		Cmd:        []string{scriptInContainer},
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
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.client.ContainerRemove(cleanupCtx, resp.ID, container.RemoveOptions{Force: true}); err != nil {
			logman.Warn("Remove temporary script container failed", "container", resp.ID, "error", err)
		}
	}()

	if err := s.ContainerFileUpload(ctx, resp.ID, scriptDir, scriptName, strings.NewReader(script)); err != nil {
		return "", fmt.Errorf("写入临时容器脚本失败: %w", err)
	}
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
	case waitResult, ok := <-statusCh:
		if !ok {
			return "", fmt.Errorf("等待容器退出失败: 状态通道已关闭")
		}
		exitCode = waitResult.StatusCode
		if waitResult.Error != nil {
			return "", fmt.Errorf("等待容器退出失败: %s", waitResult.Error.Message)
		}
	case err, ok := <-errCh:
		if !ok || err == nil {
			return "", fmt.Errorf("等待容器退出失败: 错误通道已关闭")
		}
		return "", fmt.Errorf("等待容器退出失败: %w", err)
	}

	// 读取日志
	if err := ctx.Err(); err != nil {
		return "", err
	}
	logReader, err := s.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("读取临时容器日志失败: %w", err)
	}
	defer logReader.Close()
	logs, readErr := ReadLogSnapshot(logReader, false)
	if readErr != nil {
		return "", fmt.Errorf("读取临时容器日志失败: %w", readErr)
	}
	output := strings.Join(logs, "")

	if exitCode != 0 {
		return output, fmt.Errorf("脚本退出码 %d", exitCode)
	}
	return output, nil
}
