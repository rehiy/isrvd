// Package compose 提供统一的 Compose 部署业务服务
package compose

import (
	"context"
	"fmt"
	"io"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/internal/registry"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// Service Compose 部署业务服务
type Service struct {
	docker *docker.DockerService
	swarm  *swarm.SwarmService
}

// DeployRequest 部署请求
type DeployRequest struct {
	Content   string    `json:"content" binding:"required"` // compose.yml 文件内容（必填）
	InitURL   string    `json:"initURL,omitempty"`          // 附加运行文件 zip 的下载地址（可选）
	InitFile  io.Reader `json:"-"`                          // 附加运行文件（multipart 上传，不在 JSON 中）
	ForcePull bool      `json:"forcePull,omitempty"`        // 是否强制拉取最新镜像
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
	ProjectName string `json:"projectName,omitempty"` // 实际解析到的项目名
	FileModTime int64  `json:"fileModTime,omitempty"` // compose.yml 修改时间戳（Unix 秒）；无落盘文件时为空
	Source      string `json:"source,omitempty"`      // 内容来源：file=落盘文件，runtime=运行态反推
}

// RedeployRequest 重建请求
// - ServiceName + Image 非空：从现有内容读取后更新指定服务镜像重建
// - 否则：Content 必须非空，全量重建
type RedeployRequest struct {
	Content     string `json:"content,omitempty"`     // compose.yml 内容（未指定 serviceName 时必填，用于全量重建）
	ServiceName string `json:"serviceName,omitempty"` // 目标服务名（与 image 配合，仅更新该服务镜像后重建）
	Image       string `json:"image,omitempty"`       // 新镜像（指定 serviceName 时必填）
	ForcePull   bool   `json:"forcePull,omitempty"`   // 是否强制拉取最新镜像
}

// Validate 校验重建请求的互斥参数
func (r RedeployRequest) Validate() error {
	if r.ServiceName != "" && r.Image == "" {
		return fmt.Errorf("指定服务名时 image 不能为空")
	}
	if r.ServiceName == "" && r.Content == "" {
		return fmt.Errorf("未指定服务名时 content 不能为空")
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
