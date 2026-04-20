// Package compose 业务层：组合 pkgs/compose 通用能力 + 项目约定路径。
package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/pango/logman"
	"github.com/rehiy/pango/request"

	"isrvd/pkgs/archive"
	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// safeName 校验实例名，防止路径穿越
var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// DeployService 统一的 Compose 部署业务服务
//
// 统一入口 Deploy 根据参数决定行为：
//   - ProjectName 为空：临时部署，不落盘（compose 在线编辑场景）
//   - ProjectName 非空 + Target=docker：落盘到 {ContainerRoot}/{ProjectName}，
//     可选下载 InitURL 指向的附加运行文件 zip（应用市场一键安装场景）
//   - ProjectName 非空 + Target=swarm：仅作为 compose project 名使用，不落盘
type DeployService struct {
	docker  *docker.DockerService
	compose *compose.ComposeService
	swarm   *swarm.SwarmManager
}

// NewDeployService 创建 compose 部署服务
// swarm 可为 nil，此时 target="swarm" 的请求会返回未初始化错误
func NewDeployService(d *docker.DockerService, c *compose.ComposeService, s *swarm.SwarmManager) (*DeployService, error) {
	if d == nil || c == nil {
		return nil, fmt.Errorf("docker 或 compose 服务未提供")
	}
	return &DeployService{docker: d, compose: c, swarm: s}, nil
}

// ComposeDeployTarget 部署目标
type ComposeDeployTarget string

const (
	TargetDocker ComposeDeployTarget = "docker"
	TargetSwarm  ComposeDeployTarget = "swarm"
)

// DeployRequest 统一的 compose 部署请求
//
//   - Target      : docker | swarm（必填）
//   - Content     : 完整 compose yaml 文本（必填；前端已完成变量插值）
//   - ProjectName : 可选 compose project 名；
//     Target=docker 且非空时走落盘模式：安装到 {ContainerRoot}/{ProjectName}
//   - InitURL     : 可选，仅落盘模式生效；附加运行文件 zip 下载地址
type DeployRequest struct {
	Target      ComposeDeployTarget `json:"target" binding:"required"`
	Content     string              `json:"content" binding:"required"`
	ProjectName string              `json:"projectName"`
	InitURL     string              `json:"initURL"`
}

// DeployResult 部署结果
type DeployResult struct {
	Target     ComposeDeployTarget `json:"target"`               // 部署目标
	Items      []string            `json:"items"`                // 容器名 / swarm 服务名，格式 "name (shortId)"
	InstallDir string              `json:"installDir,omitempty"` // 仅落盘模式返回
}

// Deploy 统一的 compose 部署入口
func (s *DeployService) Deploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("compose 内容不能为空")
	}

	// 判定是否走落盘模式（应用市场一键安装）
	persist := req.ProjectName != "" && req.Target == TargetDocker
	if persist {
		return s.deployPersistent(ctx, req)
	}

	// 临时部署：直接从文本加载并部署
	project, err := compose.LoadProjectFromContent(ctx, req.Content, req.ProjectName)
	if err != nil {
		return nil, err
	}
	items, err := s.deployProject(ctx, req.Target, project)
	if err != nil {
		return nil, err
	}
	return &DeployResult{Target: req.Target, Items: items}, nil
}

// deployPersistent 落盘部署（应用市场一键安装场景，仅 docker）
//
// 流程：
//  1. 在 {ContainerRoot}/{ProjectName} 下建目录
//  2. 可选：下载 InitURL 指向的 zip 并解压
//  3. 将已插值的 compose 文本落盘为 docker-compose.yml
//  4. 加载并部署为单机容器
func (s *DeployService) deployPersistent(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}
	if !safeName.MatchString(req.ProjectName) {
		return nil, fmt.Errorf("非法的实例名")
	}

	// 安装目录约定：{ContainerRoot}/{ProjectName}
	installDir := filepath.Join(root, req.ProjectName)
	if _, err := os.Stat(installDir); err == nil {
		return nil, fmt.Errorf("目录已存在：%s，请先移除或使用其它实例名", installDir)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}

	// 异常清理
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(installDir)
		}
	}()

	// 下载并解压附加运行文件
	if req.InitURL != "" {
		zipPath := filepath.Join(installDir, "init.zip")
		if _, err := request.Download(req.InitURL, zipPath, false); err != nil {
			return nil, fmt.Errorf("下载附加文件失败: %w", err)
		}
		if err := archive.NewZipper().Unzip(zipPath); err != nil {
			return nil, fmt.Errorf("解压附加文件失败: %w", err)
		}
		_ = os.Remove(zipPath)
	}

	// 将已插值的 compose 落盘（前端已完成变量替换，后端无需再做 env 处理）
	composeFile := filepath.Join(installDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(req.Content), 0644); err != nil {
		return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
	}

	// 加载并部署，项目名与实例名保持一致
	project, err := compose.LoadProject(ctx, compose.LoadOptions{
		WorkingDir:  installDir,
		ProjectName: req.ProjectName,
	})
	if err != nil {
		return nil, err
	}

	items, err := s.deployProject(ctx, TargetDocker, project)
	if err != nil {
		return nil, err
	}

	ok = true
	logman.Info("Compose app deployed",
		"name", req.ProjectName,
		"installDir", installDir,
	)
	return &DeployResult{Target: TargetDocker, Items: items, InstallDir: installDir}, nil
}

// deployProject 根据 target 分发到 docker / swarm 部署
func (s *DeployService) deployProject(ctx context.Context, target ComposeDeployTarget, project *types.Project) ([]string, error) {
	switch target {
	case TargetDocker:
		return s.compose.DeployProject(ctx, project)
	case TargetSwarm:
		if s.swarm == nil {
			return nil, fmt.Errorf("swarm manager 未初始化")
		}
		if len(project.Services) == 0 {
			return nil, fmt.Errorf("compose 文件中没有定义服务")
		}
		var services []string
		for _, svc := range project.Services {
			req, err := compose.ServiceToSwarmRequest(project, svc)
			if err != nil {
				return services, err
			}
			id, err := s.swarm.CreateService(ctx, req)
			if err != nil {
				return services, fmt.Errorf("创建服务 %s 失败: %w", req.Name, err)
			}
			if len(id) > 12 {
				id = id[:12]
			}
			services = append(services, fmt.Sprintf("%s (%s)", req.Name, id))
			logman.Info("Swarm compose service deployed", "service", svc.Name, "id", id)
		}
		return services, nil
	default:
		return nil, fmt.Errorf("不支持的部署目标: %s", target)
	}
}
