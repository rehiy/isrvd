package compose

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/archive"
	"github.com/rehiy/libgo/request"
)

const ComposeFileName = "compose.yml"

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

// ProjectNameFromProject 从 loader 结果和原始内容确定项目名。
func ProjectNameFromProject(project *types.Project, content string) (string, error) {
	if project == nil {
		return "", fmt.Errorf("project 为空")
	}
	name := project.Name
	if name == "" || name == "." {
		name = ShortHash(content)
	}
	if err := ValidateProjectName(name); err != nil {
		return "", err
	}
	return name, nil
}

// ContentProjectName 从 compose 内容解析项目名，缺失时使用内容短哈希。
func ContentProjectName(ctx context.Context, content string) (string, error) {
	project, err := LoadProjectFromContent(ctx, content, "")
	if err != nil {
		return "", err
	}
	return ProjectNameFromProject(project, content)
}

// ==================== Project loading and persistence ====================

// ProjectLoad 写入 compose.yml 并以 installDir 为 WorkingDir 加载，确保相对路径正确展开。
func ProjectLoad(ctx context.Context, name, content, installDir string) (*types.Project, error) {
	if installDir == "" {
		return LoadProjectFromContent(ctx, content, name)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, ComposeFileName), []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
	}
	return LoadProject(ctx, LoadOptions{WorkingDir: installDir, ProjectName: name})
}

// ProjectParse 解析 compose 内容（不写文件），相对路径基于 installDir 展开。
func ProjectParse(ctx context.Context, name, content, installDir string) (*types.Project, error) {
	if installDir == "" {
		return LoadProjectFromContent(ctx, content, name)
	}
	return LoadProjectFromContentInDir(ctx, content, installDir, name)
}

// ContentSave 持久化 compose.yml；bak 非空时同时写 compose.yml.bak。
func ContentSave(installDir, content, bak string) {
	if installDir == "" {
		return
	}
	_ = os.MkdirAll(installDir, 0755)
	if content != "" {
		_ = os.WriteFile(filepath.Join(installDir, ComposeFileName), []byte(content), 0644)
	}
	if bak != "" {
		_ = os.WriteFile(filepath.Join(installDir, ComposeFileName+".bak"), []byte(bak), 0644)
	}
}

// InitFilesHandle 处理附加运行文件（支持本地上传或 URL 下载），解压到 installDir。
func InitFilesHandle(installDir string, payload InitPayload) error {
	if payload.File == nil && payload.URL == "" {
		return nil
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}
	zipPath := filepath.Join(installDir, "init.zip")

	if payload.File != nil {
		if closer, ok := payload.File.(io.Closer); ok {
			defer closer.Close()
		}
		return writeAndUnzip(zipPath, payload.File)
	}

	if _, err := request.Download(payload.URL, zipPath, false); err != nil {
		return fmt.Errorf("下载附加文件失败: %w", err)
	}
	if err := archive.NewZipper().Unzip(zipPath); err != nil {
		return fmt.Errorf("解压附加文件失败: %w", err)
	}
	_ = os.Remove(zipPath)
	return nil
}

// ==================== Content mutation ====================

// UpdateServiceImage 将 compose 内容中指定服务的镜像替换为 image，返回更新后的 YAML 文本。
func UpdateServiceImage(ctx context.Context, name, content, serviceName, image string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("compose 内容不能为空")
	}
	project, err := LoadProjectFromContent(ctx, content, name)
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

func ShortHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:4])
}

func DockerContainerNameOf(svc types.ServiceConfig) string {
	if svc.ContainerName != "" {
		return svc.ContainerName
	}
	return svc.Name
}

func DockerContainerNameCandidates(projectName string, svc types.ServiceConfig) []string {
	candidates := []string{DockerContainerNameOf(svc)}
	if svc.ContainerName == "" {
		candidates = append(candidates,
			svc.Name,
			fmt.Sprintf("%s-%s-1", projectName, svc.Name),
			fmt.Sprintf("%s_%s_1", projectName, svc.Name),
		)
	}
	result := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func DockerComposeProjectName(info container.InspectResponse) string {
	if info.Config == nil || info.Config.Labels == nil {
		return ""
	}
	return info.Config.Labels[ComposeProjectLabel]
}

// ==================== File helpers ====================

func writeAndUnzip(zipPath string, r io.Reader) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("创建附加文件失败: %w", err)
	}
	defer f.Close()

	if _, err = io.Copy(f, r); err != nil {
		return fmt.Errorf("写入附加文件失败: %w", err)
	}
	if err := archive.NewZipper().Unzip(zipPath); err != nil {
		return fmt.Errorf("解压附加文件失败: %w", err)
	}
	_ = os.Remove(zipPath)
	return nil
}
