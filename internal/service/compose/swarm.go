package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

// SwarmDeploy 部署新的 Swarm Compose 项目。
func (s *Service) SwarmDeploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	// 部署前预检：不解析 env_file（避免 .env 尚未写盘时报错），
	// 返回的 project.Name 已插值+规范化；无 name 时已用内容短哈希兜底
	pre, err := compose.ProjectValidateWithoutEnvFiles(ctx, req.Content, req.EnvContent)
	if err != nil {
		return nil, err
	}
	projectName, err := compose.ProjectNameFromProject(pre, req.Content)
	if err != nil {
		return nil, err
	}

	installDir := filepath.Join(root, projectName)
	composeFile := filepath.Join(installDir, "compose.yml")
	if _, err := os.Stat(composeFile); err == nil {
		return nil, fmt.Errorf("目录 %s 已包含 compose 配置，请先移除", installDir)
	}

	_, err = os.Stat(installDir)
	installDirExists := err == nil
	initialEnvState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}

	deployed := false
	defer func() {
		if deployed {
			return
		}
		if !installDirExists {
			_ = os.RemoveAll(installDir)
			return
		}
		_ = os.Remove(composeFile)
		if err := compose.EnvStateRestore(installDir, initialEnvState); err != nil {
			logman.Warn("Restore swarm compose env after failed deploy", "name", projectName, "error", err)
		}
	}()

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}
	if err := compose.InitFilesHandle(installDir, compose.InitPayload{URL: req.InitURL, File: req.InitFile}); err != nil {
		return nil, err
	}

	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, err
	}

	project, err := compose.ProjectLoad(ctx, projectName, req.Content, installDir)
	if err != nil {
		return nil, err
	}
	if len(project.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}

	for _, svc := range project.Services {
		if _, err := s.swarm.ServiceInspect(ctx, svc.Name); err == nil {
			return nil, fmt.Errorf("服务 %s 已存在，请先移除", svc.Name)
		}
	}

	// 预拉取所有镜像（manager 节点本地校验）
	if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
		return nil, err
	}

	items, err := s.swarmServicesCreate(ctx, project)
	if err != nil {
		return nil, err
	}

	deployed = true
	logman.Info("Swarm compose deployed", "name", projectName, "dir", installDir)
	return &DeployResult{ProjectName: projectName, Items: items, InstallDir: installDir}, nil
}

// SwarmContent 读取项目的 compose.yml；文件不存在时从运行态反推。
func (s *Service) SwarmContent(ctx context.Context, name string) (string, error) {
	result, err := s.SwarmContentResult(ctx, name, false)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// SwarmContentResult 读取项目 compose.yml；forceRuntime 为 true 时跳过落盘文件，直接从运行态反推。
func (s *Service) SwarmContentResult(ctx context.Context, name string, forceRuntime bool) (*ContentResult, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	path := filepath.Join(root, name, "compose.yml")
	fileModTime := composeFileModTime(path)
	if !forceRuntime {
		if data, err := os.ReadFile(path); err == nil {
			envContent, err := compose.EnvContentRead(filepath.Join(root, name))
			if err != nil {
				return nil, err
			}
			return &ContentResult{
				Content:     string(data),
				EnvContent:  envContent,
				ProjectName: name,
				FileModTime: fileModTime,
				Source:      "file",
			}, nil
		}
	}

	raw, err := s.swarm.ServiceInspectRaw(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
	}
	project, err := compose.ProjectFromSwarmInspect(raw, filepath.Join(root, name))
	if err != nil {
		return nil, err
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return nil, err
	}
	envContent, err := compose.EnvContentRead(filepath.Join(root, name))
	if err != nil {
		return nil, err
	}
	return &ContentResult{
		Content:     string(data),
		EnvContent:  envContent,
		ProjectName: name,
		FileModTime: fileModTime,
		Source:      "runtime",
	}, nil
}

// SwarmRedeploy 重建 Swarm Compose 项目。
// 部分更新：提交哪个字段就改哪个字段，未提交的字段保持不变。
// - ServiceName+Image：仅更新指定服务的镜像后全量重建
// - Content：替换 compose.yml 后重建
// - EnvContent：替换 .env（空串即清空）后重建
func (s *Service) SwarmRedeploy(ctx context.Context, name string, req RedeployRequest) (*DeployResult, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	oldEnvState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}

	oldContent, contentErr := s.SwarmContent(ctx, name)

	// 准备新 content：未提交则沿用现有 compose 内容
	content := oldContent
	switch {
	case req.ServiceName != "":
		if contentErr != nil {
			return nil, contentErr
		}
		content, err = compose.UpdateServiceImage(ctx, name, oldContent, req.ServiceName, req.Image)
		if err != nil {
			return nil, err
		}
	case req.Content != nil:
		content = *req.Content
	default:
		// 仅更新 .env：必须能读到现有 compose 内容
		if contentErr != nil {
			return nil, contentErr
		}
	}

	// 先解析新 content 校验合法性（不写文件、不删旧实例），失败时旧服务保持运行
	newProject, err := compose.LoadProjectFromContentInDir(ctx, content, installDir, name, req.EnvContent)
	if err != nil {
		return nil, err
	}
	if len(newProject.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	// 预拉取镜像（manager 节点本地校验，验证镜像引用合法、registry 可达）
	if err := s.imagesEnsure(ctx, newProject, req.ForcePull); err != nil {
		return nil, err
	}

	s.swarmServicesRemove(ctx, name, oldContent)

	rollback := func() string {
		// 先恢复旧 .env 再回滚服务；.env 失败不阻断服务回滚
		compose.ContentSave(installDir, oldContent, "")
		envErr := compose.EnvStateRestore(installDir, oldEnvState)
		if envErr != nil {
			logman.Warn("Restore swarm compose env before rollback failed", "name", name, "error", envErr)
		}
		runtimeErr := s.swarmRollback(ctx, name, oldContent, installDir)
		if runtimeErr != nil {
			logman.Warn("Rollback swarm services failed", "name", name, "error", runtimeErr)
		}
		return formatRedeployRollbackSummary(envErr == nil, envErr, runtimeErr == nil, runtimeErr, "服务")
	}

	// 先落盘新 .env，确保 ProjectLoad 插值读取到新值（与 Deploy 流程顺序一致）
	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	project, err := compose.ProjectLoad(ctx, name, content, installDir)
	if err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	items, err := s.swarmServicesCreate(ctx, project)
	if err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	compose.ContentSave(installDir, content, oldContent)

	logman.Info("Swarm compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// ==================== 辅助函数 ====================

// swarmServicesCreate 批量创建 project 中的所有 Swarm 服务，失败时回滚已创建的服务。
// 调用前须先通过 imagesEnsure 完成预拉取。
func (s *Service) swarmServicesCreate(ctx context.Context, project *types.Project) ([]string, error) {
	if err := s.swarmEnsureNetworks(ctx, project); err != nil {
		return nil, err
	}

	var createdIDs []string
	var items []string

	rollback := func() {
		for _, id := range createdIDs {
			if err := s.swarm.ServiceAction(ctx, id, "remove", nil); err != nil {
				logman.Warn("Rollback remove service failed", "id", id, "error", err)
			}
		}
	}

	for _, svc := range project.Services {
		id, name, err := s.swarmServiceCreate(ctx, project, svc)
		if err != nil {
			rollback()
			return nil, err
		}
		createdIDs = append(createdIDs, id)
		items = append(items, fmt.Sprintf("%s (%s)", name, docker.ShortID(id)))
		logman.Info("Swarm service deployed", "service", svc.Name, "id", docker.ShortID(id))
	}
	return items, nil
}

// swarmServiceCreate 根据 compose service 创建对应 Swarm 服务。
// 不负责镜像拉取，调用前须确保镜像已存在。
func (s *Service) swarmServiceCreate(ctx context.Context, project *types.Project, svc types.ServiceConfig) (string, string, error) {
	spec, err := compose.ServiceToSwarmSpec(project, svc)
	if err != nil {
		return "", "", err
	}
	id, err := s.swarm.ServiceCreate(ctx, spec)
	if err != nil {
		return "", "", fmt.Errorf("创建服务 %s 失败: %w", spec.Name, err)
	}
	return id, spec.Name, nil
}

// swarmServicesRemove 移除 project 中的所有 Swarm 服务
func (s *Service) swarmServicesRemove(ctx context.Context, name, content string) {
	if content == "" {
		return
	}
	project, err := compose.LoadProjectFromContent(ctx, content, name)
	if err != nil {
		return
	}
	for _, svc := range project.Services {
		_ = s.swarm.ServiceAction(ctx, svc.Name, "remove", nil)
	}
}

// swarmRollback 用指定配置内容重建 Swarm 服务（回滚用）
func (s *Service) swarmRollback(ctx context.Context, name, content, installDir string) error {
	if content == "" {
		return fmt.Errorf("无可回滚的 compose 内容")
	}
	project, err := compose.ProjectParse(ctx, name, content, installDir)
	if err != nil {
		return fmt.Errorf("加载回滚配置失败: %w", err)
	}
	if _, err := s.swarmServicesCreate(ctx, project); err != nil {
		return fmt.Errorf("重建服务失败: %w", err)
	}
	return nil
}

// swarmEnsureNetworks 确保 project 中所有非 external 的网络以 overlay driver 存在
func (s *Service) swarmEnsureNetworks(ctx context.Context, project *types.Project) error {
	for key, netCfg := range project.Networks {
		if bool(netCfg.External) {
			continue
		}
		netName := netCfg.Name
		if netName == "" {
			netName = key
		}
		if _, err := s.docker.NetworkInspect(ctx, netName); err == nil {
			continue
		}
		driver := netCfg.Driver
		if driver == "" {
			driver = "overlay"
		}
		if _, err := s.docker.NetworkCreate(ctx, netName, driver, ""); err != nil {
			return fmt.Errorf("创建网络 %s 失败: %w", netName, err)
		}
		logman.Info("Swarm network created", "network", netName, "driver", driver)
	}
	return nil
}
