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

// DockerDeploy 部署新的 Docker Compose 项目。
func (s *Service) DockerDeploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
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
		cname := dockerContainerNameOf(svc)
		if _, err := s.docker.ContainerInspectRaw(ctx, cname); err == nil {
			return nil, fmt.Errorf("容器 %s 已存在，请先移除", cname)
		}
	}

	// 预拉取所有镜像，避免部署中途某个镜像拉取失败
	if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
		return nil, err
	}

	items, err := s.dockerServicesCreate(ctx, project)
	if err != nil {
		return nil, err
	}

	deployed = true
	logman.Info("Compose deployed", "name", projectName, "dir", installDir)
	return &DeployResult{ProjectName: projectName, Items: items, InstallDir: installDir}, nil
}

// DockerContentGet 读取项目的 compose.yml；文件不存在时从运行态反推。
func (s *Service) DockerContentGet(ctx context.Context, name string) (string, error) {
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

	info, err := s.docker.ContainerInspectRaw(ctx, name)
	if err != nil {
		return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
	}
	imageConfig, _ := s.docker.ImageConfig(ctx, info.Config.Image)
	project, err := compose.ProjectFromDockerInspect(info, imageConfig, filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DockerRedeploy 用新 compose 内容全量重建项目。
func (s *Service) DockerRedeploy(ctx context.Context, name, content string, forcePull bool) (*DeployResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	oldContent, _ := s.DockerContentGet(ctx, name)

	// 先解析新 content 校验合法性（不写文件、不删旧实例），失败时旧服务保持运行
	newProject, err := s.projectParse(ctx, name, content, installDir)
	if err != nil {
		return nil, err
	}
	if len(newProject.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	// 预拉取镜像，避免删除旧容器后才发现镜像不可用
	if err := s.imagesEnsure(ctx, newProject, forcePull); err != nil {
		return nil, err
	}

	s.dockerContainersRemove(ctx, name, oldContent)

	rollback := func() {
		s.dockerRollback(ctx, name, oldContent, installDir)
		s.contentSave(installDir, oldContent, "")
	}

	project, err := s.projectLoad(ctx, name, content, installDir)
	if err != nil {
		rollback()
		return nil, err
	}

	items, err := s.dockerServicesCreate(ctx, project)
	if err != nil {
		rollback()
		return nil, err
	}

	s.contentSave(installDir, content, oldContent)

	logman.Info("Compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// DockerImageRedeploy 更新项目中指定服务的镜像并重建该容器。
func (s *Service) DockerImageRedeploy(ctx context.Context, name, serviceName, image string, forcePull bool) (*DeployResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	oldContent, err := s.DockerContentGet(ctx, name)
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

	// 预拉取新镜像，避免删除旧容器后才发现镜像不可用
	if err := s.docker.ImageEnsure(ctx, newSvc.Image, forcePull); err != nil {
		return nil, fmt.Errorf("镜像 %s 不存在，拉取失败: %w", newSvc.Image, err)
	}

	oldContainerName := dockerContainerNameOf(oldSvc)
	_ = s.docker.ContainerAction(ctx, oldContainerName, "stop")
	if err := s.docker.ContainerAction(ctx, oldContainerName, "remove"); err != nil {
		s.contentSave(installDir, oldContent, "")
		return nil, fmt.Errorf("删除旧容器 %s 失败: %w", oldContainerName, err)
	}

	id, cname, err := s.dockerServiceCreate(ctx, newProject, newSvc)
	if err != nil {
		if _, _, rbErr := s.dockerServiceCreate(ctx, oldProject, oldSvc); rbErr != nil {
			logman.Warn("Compose container rollback failed", "name", name, "service", serviceName, "error", rbErr)
		}
		s.contentSave(installDir, oldContent, "")
		return nil, err
	}

	s.contentSave(installDir, newContent, oldContent)

	item := fmt.Sprintf("%s (%s)", cname, docker.ShortID(id))
	logman.Info("Compose service image redeployed", "name", name, "service", serviceName, "image", image)
	return &DeployResult{ProjectName: name, Items: []string{item}, InstallDir: installDir}, nil
}

// ==================== 辅助函数 ====================

func dockerContainerNameOf(svc types.ServiceConfig) string {
	if svc.ContainerName != "" {
		return svc.ContainerName
	}
	return svc.Name
}

// dockerServicesCreate 批量创建 project 中的所有容器，失败时回滚已创建的容器。
// 调用前须先通过 imagesEnsure 完成预拉取。
func (s *Service) dockerServicesCreate(ctx context.Context, project *types.Project) ([]string, error) {
	if err := s.dockerEnsureNetworks(ctx, project); err != nil {
		return nil, err
	}

	var createdIDs []string
	var items []string

	rollback := func() {
		for _, id := range createdIDs {
			if err := s.docker.ContainerAction(ctx, id, "remove"); err != nil {
				logman.Warn("Rollback remove container failed", "id", docker.ShortID(id), "error", err)
			}
		}
	}

	for _, svc := range project.Services {
		id, name, err := s.dockerServiceCreate(ctx, project, svc)
		if err != nil {
			rollback()
			return nil, err
		}
		createdIDs = append(createdIDs, id)
		items = append(items, fmt.Sprintf("%s (%s)", name, docker.ShortID(id)))
		logman.Info("Compose container deployed", "service", svc.Name, "container", name, "id", docker.ShortID(id))
	}
	return items, nil
}

// dockerServiceCreate 根据 compose service 创建对应 Docker 容器。
// 不负责镜像拉取，调用前须确保镜像已存在。
func (s *Service) dockerServiceCreate(ctx context.Context, project *types.Project, svc types.ServiceConfig) (string, string, error) {
	req, err := compose.ServiceToDockerRequest(project, svc)
	if err != nil {
		return "", "", err
	}
	id, err := s.docker.ContainerCreate(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("创建容器 %s 失败: %w", req.Name, err)
	}
	return id, req.Name, nil
}

// dockerContainersRemove 移除 project 中的所有 Docker 容器
func (s *Service) dockerContainersRemove(ctx context.Context, name, content string) {
	if content == "" {
		return
	}
	project, err := compose.LoadProjectFromContent(ctx, content, name)
	if err != nil {
		return
	}
	for _, svc := range project.Services {
		cname := dockerContainerNameOf(svc)
		_ = s.docker.ContainerAction(ctx, cname, "stop")
		_ = s.docker.ContainerAction(ctx, cname, "remove")
	}
}

// dockerRollback 用指定配置内容重建 Docker 容器（回滚用）
func (s *Service) dockerRollback(ctx context.Context, name, content, installDir string) {
	if content == "" {
		return
	}
	project, err := s.projectParse(ctx, name, content, installDir)
	if err != nil {
		logman.Warn("Rollback load project failed", "name", name, "error", err)
		return
	}
	if _, err := s.dockerServicesCreate(ctx, project); err != nil {
		logman.Warn("Rollback deploy failed", "name", name, "error", err)
	}
}

// dockerEnsureNetworks 确保 project 中所有 bridge 网络存在
func (s *Service) dockerEnsureNetworks(ctx context.Context, project *types.Project) error {
	for _, name := range compose.CollectNetworks(project) {
		if _, err := s.docker.NetworkInspect(ctx, name); err == nil {
			continue
		}
		if _, err := s.docker.NetworkCreate(ctx, name, "bridge", ""); err != nil {
			return fmt.Errorf("网络 %s 不存在，创建失败: %w", name, err)
		}
	}
	return nil
}
