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

	project, err := compose.LoadProjectFromContent(ctx, req.Content, "")
	if err != nil {
		return nil, err
	}
	projectName := project.Name
	if projectName == "" || projectName == "." {
		projectName = shortHash(req.Content)
	}
	if err := ValidateName(projectName); err != nil {
		return nil, err
	}

	installDir := filepath.Join(root, projectName)
	composeFile := filepath.Join(installDir, "compose.yml")
	if _, err := os.Stat(composeFile); err == nil {
		return nil, fmt.Errorf("目录 %s 已包含 compose 配置，请先移除", installDir)
	}

	_, err = os.Stat(installDir)
	installDirExists := err == nil

	deployed := false
	defer func() {
		if !deployed && !installDirExists {
			_ = os.RemoveAll(installDir)
		}
	}()

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("创建安装目录失败: %w", err)
	}
	if err := s.initFileHandle(installDir, req); err != nil {
		return nil, err
	}

	project, err = s.projectLoad(ctx, projectName, req.Content, installDir)
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

// SwarmContentGet 读取项目的 compose.yml；文件不存在时从运行态反推。
func (s *Service) SwarmContentGet(ctx context.Context, name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return "", fmt.Errorf("未配置容器数据根目录")
	}

	path := filepath.Join(root, name, "compose.yml")
	if data, err := os.ReadFile(path); err == nil {
		return string(data), nil
	}

	raw, err := s.swarm.ServiceInspectRaw(ctx, name)
	if err != nil {
		return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
	}
	project, err := compose.ProjectFromSwarmInspect(raw, filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SwarmRedeploy 用新 compose 内容全量重建项目。
func (s *Service) SwarmRedeploy(ctx context.Context, name, content string, forcePull bool) (*DeployResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	oldContent, _ := s.SwarmContentGet(ctx, name)

	// 先解析新 content 校验合法性（不写文件、不删旧实例），失败时旧服务保持运行
	newProject, err := s.projectParse(ctx, name, content, installDir)
	if err != nil {
		return nil, err
	}
	if len(newProject.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	// 预拉取镜像（manager 节点本地校验，验证镜像引用合法、registry 可达）
	if err := s.imagesEnsure(ctx, newProject, forcePull); err != nil {
		return nil, err
	}

	s.swarmServicesRemove(ctx, name, oldContent)

	rollback := func() {
		s.swarmRollback(ctx, name, oldContent, installDir)
		s.contentSave(installDir, oldContent, "")
	}

	project, err := s.projectLoad(ctx, name, content, installDir)
	if err != nil {
		rollback()
		return nil, err
	}

	items, err := s.swarmServicesCreate(ctx, project)
	if err != nil {
		rollback()
		return nil, err
	}

	s.contentSave(installDir, content, oldContent)

	logman.Info("Swarm compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// SwarmImageRedeploy 更新项目中指定服务的镜像并重建该服务。
func (s *Service) SwarmImageRedeploy(ctx context.Context, name, serviceName, image string, forcePull bool) (*DeployResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	oldContent, err := s.SwarmContentGet(ctx, name)
	if err != nil {
		return nil, err
	}
	newContent, err := updateServiceImage(ctx, name, oldContent, serviceName, image)
	if err != nil {
		return nil, err
	}

	oldProject, err := s.projectParse(ctx, name, oldContent, installDir)
	if err != nil {
		return nil, err
	}
	newProject, err := s.projectParse(ctx, name, newContent, installDir)
	if err != nil {
		return nil, err
	}
	oldSvc, err := projectServiceFind(oldProject, serviceName)
	if err != nil {
		return nil, err
	}
	newSvc, err := projectServiceFind(newProject, serviceName)
	if err != nil {
		return nil, err
	}

	// 预拉取新镜像（manager 节点本地校验）
	if err := s.docker.ImageEnsure(ctx, newSvc.Image, forcePull); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", newSvc.Image, err)
	}

	oldServiceName := oldSvc.Name
	if err := s.swarm.ServiceAction(ctx, oldServiceName, "remove", nil); err != nil {
		s.contentSave(installDir, oldContent, "")
		return nil, fmt.Errorf("删除旧服务 %s 失败: %w", oldServiceName, err)
	}

	id, sname, err := s.swarmServiceCreate(ctx, newProject, newSvc)
	if err != nil {
		if _, _, rbErr := s.swarmServiceCreate(ctx, oldProject, oldSvc); rbErr != nil {
			logman.Warn("Compose swarm service rollback failed", "name", name, "service", serviceName, "error", rbErr)
		}
		s.contentSave(installDir, oldContent, "")
		return nil, err
	}

	s.contentSave(installDir, newContent, oldContent)

	item := fmt.Sprintf("%s (%s)", sname, docker.ShortID(id))
	logman.Info("Swarm service image redeployed", "name", name, "service", serviceName, "image", image)
	return &DeployResult{ProjectName: name, Items: []string{item}, InstallDir: installDir}, nil
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
	req, err := compose.ServiceToSwarmRequest(project, svc)
	if err != nil {
		return "", "", err
	}
	id, err := s.swarm.ServiceCreate(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("创建服务 %s 失败: %w", req.Name, err)
	}
	return id, req.Name, nil
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
func (s *Service) swarmRollback(ctx context.Context, name, content, installDir string) {
	if content == "" {
		return
	}
	project, err := s.projectParse(ctx, name, content, installDir)
	if err != nil {
		logman.Warn("Rollback load project failed", "name", name, "error", err)
		return
	}
	if _, err := s.swarmServicesCreate(ctx, project); err != nil {
		logman.Warn("Rollback deploy failed", "name", name, "error", err)
	}
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
