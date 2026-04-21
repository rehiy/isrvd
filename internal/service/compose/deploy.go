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

// GetContent 统一获取 compose 文件内容
//   - target=docker：从磁盘读取 compose 文件（缺失时从运行态 inspect 反推）
//   - target=swarm ：从运行态 inspect 反推
func (s *DeployService) GetContent(ctx context.Context, target ComposeDeployTarget, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("名称不能为空")
	}
	switch target {
	case TargetDocker:
		return s.getDockerContent(ctx, name)
	case TargetSwarm:
		if s.swarm == nil {
			return "", fmt.Errorf("swarm manager 未初始化")
		}
		info, err := s.swarm.InspectService(ctx, name)
		if err != nil {
			return "", err
		}
		project, err := compose.ProjectFromSwarmInspect(info)
		if err != nil {
			return "", err
		}
		data, err := compose.ProjectToYAML(project)
		return string(data), err
	default:
		return "", fmt.Errorf("不支持的目标: %s", target)
	}
}

// getDockerContent 读取 docker compose 文件内容；
// 快照缺失时自动从运行态 inspect 反推并写入，然后返回内容。
func (s *DeployService) getDockerContent(ctx context.Context, name string) (string, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return "", fmt.Errorf("未配置容器数据根目录")
	}
	path := filepath.Join(root, name, composeFileName)

	if _, err := os.Stat(path); err != nil {
		info, err := s.docker.InspectContainer(ctx, name)
		if err != nil {
			return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
		}
		project, err := compose.ProjectFromInspect(info)
		if err != nil {
			return "", err
		}
		data, err := compose.ProjectToYAML(project)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", fmt.Errorf("写入 compose 文件失败: %w", err)
		}
		return string(data), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 compose 文件失败: %w", err)
	}
	return string(data), nil
}

const composeFileName = "docker-compose.yml"

// safeName 校验实例名，防止路径穿越
var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// DeployService 统一的 Compose 部署业务服务
//
// 统一入口 Deploy 根据 Target 决定行为：
//   - Target=docker：落盘到 {ContainerRoot}/{ProjectName}，可选下载 InitURL
//     指向的附加运行文件 zip；实例名即 compose project 名
//   - Target=swarm ：仅作为 compose project 名使用，不落盘（由 swarm 集群管理）
type DeployService struct {
	compose *compose.ComposeService
	docker  *docker.DockerService
	swarm   *swarm.SwarmService
}

// NewDeployService 创建 compose 部署服务
// swarm 可为 nil，此时 target="swarm" 的请求会返回未初始化错误
func NewDeployService(d *docker.DockerService, c *compose.ComposeService, s *swarm.SwarmService) (*DeployService, error) {
	if d == nil || c == nil {
		return nil, fmt.Errorf("docker 或 compose 服务未提供")
	}
	return &DeployService{compose: c, docker: d, swarm: s}, nil
}

// ComposeDeployTarget 部署目标
type ComposeDeployTarget string

const (
	TargetDocker ComposeDeployTarget = "docker"
	TargetSwarm  ComposeDeployTarget = "swarm"
)

// DeployDockerRequest docker compose 部署请求
//
//   - Content     : 完整 compose yaml 文本（必填；前端已完成变量插值）
//   - ProjectName : 实例名（必填），同时作为 compose project 名；落盘到 {ContainerRoot}/{ProjectName}
//   - InitURL     : 可选；附加运行文件 zip 下载地址
type DeployDockerRequest struct {
	Content     string `json:"content" binding:"required"`
	ProjectName string `json:"projectName" binding:"required"`
	InitURL     string `json:"initURL"`
}

// DeploySwarmRequest swarm compose 部署请求
//
//   - Content     : 完整 compose yaml 文本（必填；前端已完成变量插值）
//   - ProjectName : 实例名（必填），同时作为 compose project 名
type DeploySwarmRequest struct {
	Content     string `json:"content" binding:"required"`
	ProjectName string `json:"projectName" binding:"required"`
}

// DeployResult 部署结果
type DeployResult struct {
	Target     ComposeDeployTarget `json:"target"`               // 部署目标
	Items      []string            `json:"items"`                // 容器名 / swarm 服务名，格式 "name (shortId)"
	InstallDir string              `json:"installDir,omitempty"` // 仅 docker 落盘时返回
}

// DeployDocker docker compose 部署入口
func (s *DeployService) DeployDocker(ctx context.Context, req DeployDockerRequest) (*DeployResult, error) {
	if !safeName.MatchString(req.ProjectName) {
		return nil, fmt.Errorf("非法的实例名")
	}
	return s.deployDocker(ctx, req)
}

// DeploySwarm swarm compose 部署入口
func (s *DeployService) DeploySwarm(ctx context.Context, req DeploySwarmRequest) (*DeployResult, error) {
	if !safeName.MatchString(req.ProjectName) {
		return nil, fmt.Errorf("非法的实例名")
	}
	return s.deploySwarm(ctx, req)
}

// RedeployRequest compose 重建请求
type RedeployRequest struct {
	Content string `json:"content" binding:"required"`
}

// validateRedeploy 校验重建请求公共参数
func (s *DeployService) validateRedeploy(name, content string) error {
	if content == "" {
		return fmt.Errorf("compose 内容不能为空")
	}
	if !safeName.MatchString(name) {
		return fmt.Errorf("非法的实例名")
	}
	return nil
}

// RedeployDocker 更新已有容器的 compose 文件并重建
// 先停止并删除旧容器，再用新内容重新部署
func (s *DeployService) RedeployDocker(ctx context.Context, name string, req RedeployRequest) (*DeployResult, error) {
	if err := s.validateRedeploy(name, req.Content); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	// 停止并删除旧容器（忽略不存在的错误）
	_ = s.docker.ContainerAction(ctx, name, "stop")
	_ = s.docker.ContainerAction(ctx, name, "remove")

	// 如果有安装目录，更新 compose 文件
	if installDir != "" {
		if err := os.MkdirAll(installDir, 0755); err != nil {
			return nil, fmt.Errorf("创建安装目录失败: %w", err)
		}
		composeFile := filepath.Join(installDir, "docker-compose.yml")
		if err := os.WriteFile(composeFile, []byte(req.Content), 0644); err != nil {
			return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
		}
	}

	// 加载并重新部署
	project, err := compose.LoadProjectFromContent(ctx, req.Content, name)
	if err != nil {
		return nil, err
	}

	items, err := s.compose.DeployProject(ctx, project)
	if err != nil {
		return nil, err
	}

	logman.Info("Compose app redeployed", "name", name)
	return &DeployResult{Target: TargetDocker, Items: items, InstallDir: installDir}, nil
}

// deployDocker 单机 docker 部署：落盘到 {ContainerRoot}/{ProjectName}，
// 可选下载并解压 InitURL 指向的附加运行文件 zip。
func (s *DeployService) deployDocker(ctx context.Context, req DeployDockerRequest) (*DeployResult, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
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

	items, err := s.compose.DeployProject(ctx, project)
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

// deploySwarm swarm 部署：不落盘，仅将 ProjectName 作为 compose project 名
// 用于生成服务默认前缀，实际状态由 swarm 集群管理。
func (s *DeployService) deploySwarm(ctx context.Context, req DeploySwarmRequest) (*DeployResult, error) {
	if s.swarm == nil {
		return nil, fmt.Errorf("swarm manager 未初始化")
	}

	project, err := compose.LoadProjectFromContent(ctx, req.Content, req.ProjectName)
	if err != nil {
		return nil, err
	}
	if len(project.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}

	items, err := s.deploySwarmProject(ctx, project)
	if err != nil {
		return nil, err
	}
	return &DeployResult{Target: TargetSwarm, Items: items}, nil
}

// deploySwarmProject 将 compose project 中的服务逐个创建为 swarm 服务
func (s *DeployService) deploySwarmProject(ctx context.Context, project *types.Project) ([]string, error) {
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
}

// RedeploySwarm 更新 swarm 服务：先删除旧服务，再用新 compose 内容重建
// name 用于匹配并删除旧服务（按 {name}_{serviceName} 前缀匹配）
func (s *DeployService) RedeploySwarm(ctx context.Context, name string, req RedeployRequest) (*DeployResult, error) {
	if err := s.validateRedeploy(name, req.Content); err != nil {
		return nil, err
	}
	if s.swarm == nil {
		return nil, fmt.Errorf("swarm manager 未初始化")
	}

	project, err := compose.LoadProjectFromContent(ctx, req.Content, name)
	if err != nil {
		return nil, err
	}
	if len(project.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}

	// 删除旧服务（按服务名匹配，忽略不存在的错误）
	for _, svc := range project.Services {
		_ = s.swarm.ServiceAction(ctx, svc.Name, "remove", nil)
	}

	items, err := s.deploySwarmProject(ctx, project)
	if err != nil {
		return nil, err
	}

	logman.Info("Swarm compose redeployed", "name", name)
	return &DeployResult{Target: TargetSwarm, Items: items}, nil
}
