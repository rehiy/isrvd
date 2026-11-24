package compose

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/request"

	"isrvd/pkgs/helper"
)

const (
	ComposeFileName = "compose.yml"
	// EnvFileName 项目 .env 文件名。
	EnvFileName = ".env"
	envFileMode = 0644
)

var safeProjectName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// InitPayload 描述 Compose 部署前需要解压到安装目录的附加运行文件。
type InitPayload struct {
	URL  string    // 附加运行文件 zip 的下载地址
	File io.Reader // 附加运行文件流（multipart 上传）
}

// ==================== Project names ====================

// ValidateProjectName 校验 Compose 项目名，防止路径逃逸。
func ValidateProjectName(name string) error {
	if name == "" || !safeProjectName.MatchString(name) {
		return fmt.Errorf("非法的项目名称: %s", name)
	}
	return nil
}

// ==================== Project loading and persistence ====================

// ProjectSave 保存已预检的 compose.yml。
func ProjectSave(content, installDir string) error {
	if installDir == "" {
		return fmt.Errorf("缺少 compose 项目工作目录")
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, ComposeFileName), []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 compose 文件失败: %w", err)
	}
	return nil
}

// EnvSave 以 env 为准写入 .env，空串表示清空内容。
func EnvSave(installDir, env string) error {
	if installDir == "" {
		return nil
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建环境文件目录失败: %w", err)
	}
	path := filepath.Join(installDir, EnvFileName)
	previous, err := os.Lstat(path)
	if err == nil {
		if !previous.Mode().IsRegular() {
			return fmt.Errorf(".env 必须是普通文件")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查 .env 失败: %w", err)
	}
	// 在私有目录内完整写入，创建模式仍受 umask 限制。
	stageDir, err := os.MkdirTemp(installDir, ".isrvd-env-*")
	if err != nil {
		return fmt.Errorf("创建 .env 临时文件失败: %w", err)
	}
	defer os.Remove(stageDir)
	temp, err := os.OpenFile(filepath.Join(stageDir, EnvFileName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, envFileMode)
	if err != nil {
		return fmt.Errorf("创建 .env 临时文件失败: %w", err)
	}
	defer os.Remove(temp.Name())
	_, writeErr := io.WriteString(temp, env)
	if writeErr == nil && previous != nil {
		writeErr = helper.PreserveFileOwner(temp, previous)
		if writeErr == nil {
			writeErr = temp.Chmod(previous.Mode().Perm())
		}
	}
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return fmt.Errorf("写入 .env 失败: %w", err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("发布 .env 失败: %w", err)
	}
	return nil
}

// EnvEnsure 确保 .env 存在（防止 env_file: .env 因缺文件报错），不修改已有内容。
func EnvEnsure(installDir string) error {
	if installDir == "" {
		return nil
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建环境文件目录失败: %w", err)
	}
	envPath := filepath.Join(installDir, EnvFileName)
	if info, err := os.Lstat(envPath); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf(".env 必须是普通文件")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查 .env 失败: %w", err)
	}
	file, err := os.OpenFile(envPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, envFileMode)
	if errors.Is(err, os.ErrExist) {
		return EnvEnsure(installDir)
	}
	if err != nil {
		return fmt.Errorf("创建 .env 失败: %w", err)
	}
	return file.Close()
}

// EnvState 记录 .env 的原始内容及是否存在，用于失败回滚。
type EnvState struct {
	Content string // .env 原始内容
	Exists  bool   // .env 是否存在
}

// EnvStateRead 读取可精确恢复的 .env 状态。
func EnvStateRead(installDir string) (EnvState, error) {
	if installDir == "" {
		return EnvState{}, nil
	}
	data, err := os.ReadFile(filepath.Join(installDir, EnvFileName))
	if errors.Is(err, os.ErrNotExist) {
		return EnvState{}, nil
	}
	if err != nil {
		return EnvState{}, fmt.Errorf("读取 .env 失败: %w", err)
	}
	return EnvState{Content: string(data), Exists: true}, nil
}

// EnvStateRestore 精确恢复 .env，包括空文件和原本不存在两种状态。
func EnvStateRestore(installDir string, state EnvState) error {
	if installDir == "" {
		return nil
	}
	path := filepath.Join(installDir, EnvFileName)
	if !state.Exists {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("恢复 .env 缺失状态失败: %w", err)
		}
		return nil
	}
	return EnvSave(installDir, state.Content)
}

// EnvApply 统一 .env 写盘/兜底：显式内容非 nil 则以其为准写盘（空串即清空），nil 则确保 .env 存在（兜底空文件）。
func EnvApply(installDir string, envContent *string) error {
	if envContent != nil {
		return EnvSave(installDir, *envContent)
	}
	return EnvEnsure(installDir)
}

// InitFilesHandle 下载或读取附加包，解压新增文件并返回失败时的清理函数。
func InitFilesHandle(ctx context.Context, installDir string, payload InitPayload) (func() error, error) {
	if payload.File == nil && payload.URL == "" {
		return nil, nil
	}
	const maximumBytes = 512 << 20
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var source = payload.File
	if source == nil {
		response, err := (&request.Client{Method: http.MethodGet, Url: payload.URL}).DoContext(ctx, http.DefaultClient)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("下载附加文件失败: HTTP %d", response.StatusCode)
		}
		source = response.Body
	}
	temp, err := os.CreateTemp("", "compose-init-*.zip")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp.Name())
	n, err := io.Copy(temp, io.LimitReader(helper.ContextReader{Context: ctx, Reader: source}, maximumBytes+1))
	if err := errors.Join(err, temp.Close()); err != nil {
		return nil, err
	}
	if n > maximumBytes {
		return nil, fmt.Errorf("附加文件大小超过限制")
	}
	return helper.ExtractZIP(ctx, temp.Name(), installDir, false, maximumBytes)
}

// ==================== Content mutation ====================

// ServiceImageUpdate 将 compose 内容中指定服务的镜像替换为 image，返回更新后的 YAML 文本。
func ServiceImageUpdate(ctx context.Context, name, content, serviceName, image, installDir string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("compose 内容不能为空")
	}
	if installDir == "" {
		return "", fmt.Errorf("缺少 compose 项目工作目录")
	}
	project, err := loadProjectContent(ctx, content, installDir, name, false, nil, true)
	if err != nil {
		return "", err
	}
	if len(project.Services) == 0 {
		return "", fmt.Errorf("compose 文件中没有定义服务")
	}

	matched := false
	for key, svc := range project.Services {
		if svc.Name == serviceName {
			svc.Image = image
			project.Services[key] = svc
			matched = true
			break
		}
	}
	if !matched {
		return "", fmt.Errorf("compose 服务 %s 不存在", serviceName)
	}

	data, err := ProjectToYAML(project)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ==================== Runtime naming helpers ====================

func DockerComposeProjectName(info container.InspectResponse) string {
	if info.Config == nil || info.Config.Labels == nil {
		return ""
	}
	return info.Config.Labels[ComposeProjectLabel]
}

// ─── 辅助函数 ───
