// Package compose 提供统一的 Compose 部署业务服务
package compose

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"

	"isrvd/server/i18n"
)

// Service Compose 部署业务服务
type Service struct {
	docker       *docker.DockerService
	swarm        *swarm.SwarmService
	deploymentMu sync.Mutex
	historyMu    sync.Mutex
}

// NewService 创建 Compose 业务服务。dockerRaw/swarmRaw 由调用方注入，swarmRaw 可为 nil。
func NewService(dockerRaw *docker.DockerService, swarmRaw *swarm.SwarmService) (*Service, error) {
	if dockerRaw == nil {
		return nil, fmt.Errorf("docker 服务未初始化")
	}
	return &Service{docker: dockerRaw, swarm: swarmRaw}, nil
}

// DeployRequest 部署请求
// EnvContent 三态：nil 保留附加文件解压出的 .env；非 nil 时以其为准写盘（空串即清空）
type DeployRequest struct {
	Content    string    `json:"content" binding:"required"` // compose.yml 文件内容（必填）
	EnvContent *string   `json:"envContent,omitempty"`       // .env 内容（KEY=VALUE）；nil 表示保留附加文件解压出的 .env，非 nil 时以其为准写盘（空串即清空）
	InitURL    string    `json:"initURL,omitempty"`          // 附加运行文件 zip 的下载地址（可选）
	InitFile   io.Reader `json:"-"`                          // 附加运行文件（multipart 上传，不在 JSON 中）
	ForcePull  bool      `json:"forcePull,omitempty"`        // 是否强制拉取最新镜像
}

// DeployResult 部署结果
type DeployResult struct {
	ProjectName string   `json:"projectName"`          // 实际使用的项目名
	Items       []string `json:"items"`                // 创建或重建的容器/服务列表
	InstallDir  string   `json:"installDir,omitempty"` // 项目落盘目录
}

// ConfigDetail Compose 配置详情。
type ConfigDetail struct {
	Content      string `json:"content"`               // compose.yml 文本
	EnvContent   string `json:"envContent,omitempty"`  // .env 文本（KEY=VALUE）；无落盘文件时为空
	ProjectName  string `json:"projectName,omitempty"` // 实际解析到的项目名
	FileModTime  int64  `json:"fileModTime,omitempty"` // compose.yml 修改时间戳（Unix 秒）；无落盘文件时为空
	Source       string `json:"source,omitempty"`      // 内容来源：file=落盘文件，runtime=运行态反推
	envExists    bool   // 区分缺失和空文件，供回滚使用
	standaloneID string // 由单个容器或服务反推配置时明确选中的 ID
}

// RedeployRequest 重建请求
// 部分更新语义——提交哪个字段就改哪个字段，未提交的字段保持不变；ServiceName + Image 用于仅更新指定服务镜像，与 Content / EnvContent 互斥。
// EnvContent 三态：nil 保留现有 .env，空串清空，非空覆盖
type RedeployRequest struct {
	Content     *string `json:"content,omitempty"`     // compose.yml 内容；nil 表示沿用现有 compose.yml，提交空串报错
	EnvContent  *string `json:"envContent,omitempty"`  // .env 内容（KEY=VALUE）；nil 表示保留现有 .env，空串表示清空，非空表示覆盖
	ServiceName string  `json:"serviceName,omitempty"` // 目标服务名（与 image 配合，仅更新该服务镜像后重建）
	Image       string  `json:"image,omitempty"`       // 新镜像（指定 serviceName 时必填）
	ForcePull   bool    `json:"forcePull,omitempty"`   // 是否强制拉取最新镜像
}

// Validate 校验重建请求的互斥参数
func (r RedeployRequest) Validate() error {
	if r.ServiceName != "" && r.Image == "" {
		return fmt.Errorf("指定服务名时 image 不能为空")
	}
	if r.ServiceName != "" && r.Content != nil {
		return fmt.Errorf("指定服务名时不能同时提交 content")
	}
	if r.ServiceName != "" && r.EnvContent != nil {
		return fmt.Errorf("指定服务名时不能同时提交 envContent")
	}
	if r.Content != nil && *r.Content == "" {
		return fmt.Errorf("content 不能为空字符串")
	}
	if r.ServiceName == "" && r.Content == nil && r.EnvContent == nil {
		return fmt.Errorf("未指定服务名时 content 与 envContent 至少需提交一项")
	}
	return nil
}

func (s *Service) beginDeployment(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.deploymentMu.TryLock() {
		return fmt.Errorf("已有 Compose 部署正在执行，请稍后重试")
	}
	return nil
}

// CheckAvailability 检测 Compose 可用性（等价于 Docker 可用）
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.docker == nil {
		return false
	}
	_, err := s.docker.Info(ctx)
	return err == nil
}

// imagesEnsure 预拉取单机 Compose 项目的镜像，避免删除旧容器后才发现镜像不可用。
// forcePull 为 true 时，无论本地是否存在都重新拉取。
func (s *Service) imagesEnsure(ctx context.Context, project *types.Project, forcePull bool) error {
	for _, svc := range project.Services {
		if svc.Image == "" {
			continue
		}
		if err := s.docker.ImageEnsure(ctx, svc.Image, forcePull); err != nil {
			return fmt.Errorf("镜像 %s 不存在，拉取失败: %w", svc.Image, err)
		}
	}
	return nil
}

// projectDeploy 统一准备部署目录与配置，并在部署失败时还原原有 .env。
func (s *Service) projectDeploy(ctx context.Context, req DeployRequest, rollbackMessage string, deploy func(*types.Project) ([]string, error)) (*DeployResult, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}
	// .env 尚未写盘，预检时跳过 env_file。
	pre, err := compose.ProjectValidateWithoutEnvFiles(ctx, req.Content, req.EnvContent)
	if err != nil {
		return nil, err
	}
	projectName := pre.Name
	if err := compose.ValidateProjectName(projectName); err != nil {
		return nil, err
	}
	installDir := filepath.Join(root, projectName)
	composeFile := filepath.Join(installDir, compose.ComposeFileName)
	if _, err := os.Lstat(composeFile); err == nil {
		return nil, fmt.Errorf("目录 %s 已包含 compose 配置，请先移除", installDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	initialEnvState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}
	var deployed, dirCreated, composeCreated, envApplied bool
	var removeAttachments func() error
	defer func() {
		if deployed {
			return
		}
		if removeAttachments != nil {
			if err := removeAttachments(); err != nil {
				logman.Warn("Remove deployment attachments failed", "name", projectName, "error", err)
			}
		}
		if composeCreated {
			_ = os.Remove(composeFile)
		}
		if envApplied {
			if err := compose.EnvStateRestore(installDir, initialEnvState); err != nil {
				logman.Warn(rollbackMessage, "name", projectName, "error", err)
			}
		}
		if dirCreated {
			_ = os.Remove(installDir)
		}
	}()
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}
	if err := os.Mkdir(installDir, 0755); err == nil {
		dirCreated = true
	} else if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}
	removeAttachments, err = compose.InitFilesHandle(ctx, installDir, compose.InitPayload{URL: req.InitURL, File: req.InitFile})
	if err != nil {
		return nil, err
	}
	envApplied = true
	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(composeFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
	}
	composeCreated = true
	_, err = file.WriteString(req.Content)
	if err := errors.Join(err, file.Close()); err != nil {
		return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
	}
	project, err := compose.LoadProject(ctx, installDir, projectName)
	if err != nil {
		return nil, err
	}
	if len(project.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	items, err := deploy(project)
	if err != nil {
		return nil, err
	}
	deployed = true
	return &DeployResult{ProjectName: projectName, Items: items, InstallDir: installDir}, nil
}

// resolveRedeployContent 解析待部署内容并返回项目，供预览与实际部署共用。
func (s *Service) resolveRedeployContent(ctx context.Context, name, installDir, oldContent string, req RedeployRequest) (string, *types.Project, error) {
	content := oldContent
	switch {
	case req.ServiceName != "":
		if !compose.HasProjectName(oldContent) {
			return "", nil, fmt.Errorf("compose 项目必须指定非空 name")
		}
		var err error
		content, err = compose.ServiceImageUpdate(ctx, name, oldContent, req.ServiceName, req.Image, installDir)
		if err != nil {
			return "", nil, err
		}
		content = withGeneratedHeader(content)
	case req.Content != nil:
		content = *req.Content
	}

	if !compose.HasProjectName(content) {
		return "", nil, fmt.Errorf("compose 项目必须指定非空 name")
	}
	envContent := req.EnvContent
	if envContent == nil {
		// EnvApply 会为没有 .env 的项目创建空文件。
		if _, err := os.Stat(filepath.Join(installDir, compose.EnvFileName)); errors.Is(err, os.ErrNotExist) {
			envContent = new("")
		}
	}
	project, err := compose.LoadProjectFromContentInDir(ctx, content, installDir, "", envContent)
	if err != nil {
		return "", nil, err
	}
	if project.Name != name {
		return "", nil, fmt.Errorf("compose 项目名必须为 %s", name)
	}
	if len(project.Services) == 0 {
		return "", nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	return content, project, nil
}

// ─── 辅助函数 ───

const composeCleanupTimeout = 30 * time.Second

func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), composeCleanupTimeout)
}

// generatedHeader 由 isrvd 序列化生成的 compose 文件头。
const generatedHeader = "# Auto-generated by iSrvd\n\n"

// withGeneratedHeader 为序列化生成的 compose YAML 添加文件头。
func withGeneratedHeader(content string) string {
	return generatedHeader + content
}

// inspectComposeConfig 优先读取落盘配置，按需反推运行态，再读取 .env 并组装详情。
func inspectComposeConfig(installDir, projectName string, forceRuntime bool, runtimeContent func() (string, error)) (*ConfigDetail, error) {
	path := filepath.Join(installDir, compose.ComposeFileName)
	fileModTime := composeFileModTime(path)
	var content string
	source := "runtime"
	if !forceRuntime {
		if data, err := os.ReadFile(path); err == nil {
			content = string(data)
			source = "file"
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("读取 compose 文件失败: %w", err)
		}
	}
	if source == "runtime" {
		var err error
		content, err = runtimeContent()
		if err != nil {
			return nil, err
		}
	}
	env, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}
	return &ConfigDetail{
		Content:     content,
		EnvContent:  env.Content,
		envExists:   env.Exists,
		ProjectName: projectName,
		FileModTime: fileModTime,
		Source:      source,
	}, nil
}

func composeFileModTime(path string) int64 {
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return st.ModTime().Unix()
	}
	return 0
}

// restore 恢复重部署前的项目文件，保留空文件与原本不存在的区别。
func (detail *ConfigDetail) restore(installDir string) (composeErr, envErr error) {
	path := filepath.Join(installDir, compose.ComposeFileName)
	if detail.Source == "file" {
		composeErr = os.WriteFile(path, []byte(detail.Content), 0644)
	} else if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		composeErr = err
	}
	envErr = compose.EnvStateRestore(installDir, compose.EnvState{Content: detail.EnvContent, Exists: detail.envExists})
	return composeErr, envErr
}

func rollbackResult(ctx context.Context, resource string, err error) string {
	resource = i18n.TC(ctx, resource)
	if err != nil {
		return fmt.Sprintf(i18n.TC(ctx, "%s 回滚失败（%v）"), resource, i18n.TC(ctx, err.Error()))
	}
	return fmt.Sprintf(i18n.TC(ctx, "%s 回滚成功"), resource)
}

func wrapRedeployError(ctx context.Context, err error, results ...string) error {
	return fmt.Errorf("%w%s%s", err, i18n.TC(ctx, "；回滚："), strings.Join(results, i18n.TC(ctx, "，")))
}
