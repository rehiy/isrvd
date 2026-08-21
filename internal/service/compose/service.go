// Package compose 提供统一的 Compose 部署业务服务
package compose

import (
	"context"
	"fmt"
	"io"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/internal/registry"
	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// Service Compose 部署业务服务
type Service struct {
	docker *docker.DockerService
	swarm  *swarm.SwarmService
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

// ContentResult Compose 配置读取结果。
type ContentResult struct {
	Content     string `json:"content"`               // compose.yml 文本
	EnvContent  string `json:"envContent,omitempty"`  // .env 文本（KEY=VALUE）；无落盘文件时为空
	ProjectName string `json:"projectName,omitempty"` // 实际解析到的项目名
	FileModTime int64  `json:"fileModTime,omitempty"` // compose.yml 修改时间戳（Unix 秒）；无落盘文件时为空
	Source      string `json:"source,omitempty"`      // 内容来源：file=落盘文件，runtime=运行态反推
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

// NewService 创建 Compose 业务服务
func NewService() (*Service, error) {
	if registry.DockerService == nil {
		return nil, fmt.Errorf("docker 服务未初始化")
	}
	return &Service{docker: registry.DockerService, swarm: registry.SwarmService}, nil
}

// CheckAvailability 检测 Compose 可用性（等价于 Docker 可用）
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.docker == nil {
		return false
	}
	_, err := s.docker.Info(ctx)
	return err == nil
}

// imagesEnsure 预拉取 project 中所有 service 的镜像，避免删除旧实例后才发现镜像不可用。
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

// prepareRedeployContent 合并部分更新、校验新配置并预拉取所需镜像。
func (s *Service) prepareRedeployContent(ctx context.Context, name, installDir, oldContent string, contentErr error, req RedeployRequest) (string, error) {
	content := oldContent
	switch {
	case req.ServiceName != "":
		if contentErr != nil {
			return "", contentErr
		}
		var err error
		content, err = compose.UpdateServiceImage(ctx, name, oldContent, req.ServiceName, req.Image, installDir)
		if err != nil {
			return "", err
		}
	case req.Content != nil:
		content = *req.Content
	case contentErr != nil:
		// 仅更新 .env 时必须能读取现有 compose 内容。
		return "", contentErr
	}

	project, err := compose.LoadProjectFromContent(ctx, content, installDir, name, req.EnvContent)
	if err != nil {
		return "", err
	}
	if len(project.Services) == 0 {
		return "", fmt.Errorf("compose 文件中没有定义服务")
	}
	if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
		return "", err
	}
	return content, nil
}

// formatRedeployRollbackSummary 汇总重建失败后的回滚结果，供前端展示。
// runtimeLabel 为「容器」或「服务」。
func formatRedeployRollbackSummary(envErr, runtimeErr error, runtimeLabel string) string {
	envPart := ".env 回滚成功"
	if envErr != nil {
		envPart = fmt.Sprintf(".env 回滚失败（%v）", envErr)
	}
	runtimePart := runtimeLabel + "回滚成功"
	if runtimeErr != nil {
		runtimePart = fmt.Sprintf("%s回滚失败（%v）", runtimeLabel, runtimeErr)
	}
	return envPart + "，" + runtimePart
}

// wrapRedeployError 将原始重建错误与回滚摘要一并返回。
func wrapRedeployError(err error, rollbackSummary string) error {
	return fmt.Errorf("%w；回滚：%s", err, rollbackSummary)
}
