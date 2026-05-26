package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/docker-image-spec/specs-go/v1"
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
	content, _, err := s.dockerContentGet(ctx, name)
	return content, err
}

func (s *Service) dockerContentGet(ctx context.Context, name string) (string, string, error) {
	if err := ValidateName(name); err != nil {
		return "", "", err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return "", "", fmt.Errorf("未配置容器数据根目录")
	}

	projectName := s.dockerProjectName(ctx, name, root)
	path := filepath.Join(root, projectName, "compose.yml")
	if data, err := os.ReadFile(path); err == nil {
		return string(data), projectName, nil
	}

	if content, ok, err := s.dockerProjectContentFromContainers(ctx, projectName, root); ok || err != nil {
		return content, projectName, err
	}

	info, err := s.docker.ContainerInspectRaw(ctx, name)
	if err != nil {
		return "", "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
	}
	imageConfig, _ := s.docker.ImageConfig(ctx, info.Config.Image)
	project, err := compose.ProjectFromDockerInspect(info, imageConfig, filepath.Join(root, name))
	if err != nil {
		return "", "", err
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return "", "", err
	}
	return string(data), name, nil
}

// DockerRedeploy 重建 Docker Compose 项目。
// req.ServiceName + req.Image 非空时：仅更新指定服务的镜像后全量重建；
// 否则：用 req.Content 全量重建。
func (s *Service) DockerRedeploy(ctx context.Context, name string, req RedeployRequest) (*DeployResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	root := s.docker.ContainerRoot()
	if root != "" {
		name = s.dockerProjectName(ctx, name, root)
	}
	installDir := ""
	if root != "" {
		installDir = filepath.Join(root, name)
	}

	// 准备新 content：单服务镜像更新 or 全量替换
	content := req.Content
	if req.ServiceName != "" {
		oldContent, _, err := s.dockerContentGet(ctx, name)
		if err != nil {
			return nil, err
		}
		content, err = updateServiceImage(ctx, name, oldContent, req.ServiceName, req.Image)
		if err != nil {
			return nil, err
		}
	}

	oldContent, _, _ := s.dockerContentGet(ctx, name)

	// 先解析新 content 校验合法性（不写文件、不删旧实例），失败时旧服务保持运行
	newProject, err := s.projectParse(ctx, name, content, installDir)
	if err != nil {
		return nil, err
	}
	if len(newProject.Services) == 0 {
		return nil, fmt.Errorf("compose 文件中没有定义服务")
	}
	// 预拉取镜像，避免删除旧容器后才发现镜像不可用
	if err := s.imagesEnsure(ctx, newProject, req.ForcePull); err != nil {
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

// ==================== 辅助函数 ====================

func dockerContainerNameOf(svc types.ServiceConfig) string {
	if svc.ContainerName != "" {
		return svc.ContainerName
	}
	return svc.Name
}

func dockerContainerNameCandidates(projectName string, svc types.ServiceConfig) []string {
	candidates := []string{dockerContainerNameOf(svc)}
	if svc.ContainerName == "" {
		candidates = append(candidates,
			svc.Name,
			fmt.Sprintf("%s-%s-1", projectName, svc.Name),
			fmt.Sprintf("%s_%s_1", projectName, svc.Name),
		)
	}
	result := candidates[:0]
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
	removed := map[string]struct{}{}
	remove := func(id string) {
		if id == "" {
			return
		}
		if _, ok := removed[id]; ok {
			return
		}
		removed[id] = struct{}{}
		_ = s.docker.ContainerAction(ctx, id, "stop")
		_ = s.docker.ContainerAction(ctx, id, "remove")
	}

	if content != "" {
		project, err := compose.LoadProjectFromContent(ctx, content, name)
		if err == nil {
			for _, svc := range project.Services {
				for _, cname := range dockerContainerNameCandidates(name, svc) {
					remove(cname)
				}
			}
		}
	}

	infos, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, name)
	if err != nil {
		logman.Warn("List compose project containers failed", "name", name, "error", err)
		return
	}
	for _, info := range infos {
		remove(info.ID)
	}
}

func (s *Service) dockerProjectName(ctx context.Context, name, root string) string {
	if root != "" {
		if _, err := os.Stat(filepath.Join(root, name, "compose.yml")); err == nil {
			return name
		}
	}
	if infos, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, name); err == nil && len(infos) > 0 {
		return name
	}
	if info, err := s.docker.ContainerInspectRaw(ctx, name); err == nil {
		if projectName := dockerComposeProjectName(info); projectName != "" {
			if err := ValidateName(projectName); err == nil {
				return projectName
			}
			logman.Warn("Ignore invalid compose project label", "container", name, "project", projectName)
		}
	}
	return name
}

func (s *Service) dockerProjectContentFromContainers(ctx context.Context, projectName, root string) (string, bool, error) {
	infos, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, projectName)
	if err != nil {
		return "", false, err
	}
	if len(infos) == 0 {
		return "", false, nil
	}

	configs := make(map[string]*v1.DockerOCIImageConfig, len(infos))
	for _, info := range infos {
		if info.Config == nil || info.Config.Image == "" {
			continue
		}
		if _, ok := configs[info.Config.Image]; ok {
			continue
		}
		if cfg, err := s.docker.ImageConfig(ctx, info.Config.Image); err == nil {
			configs[info.Config.Image] = cfg
		}
	}
	project, err := compose.ProjectFromDockerInspects(infos, configs, projectName, filepath.Join(root, projectName))
	if err != nil {
		return "", true, err
	}
	data, err := compose.ProjectToYAML(project)
	if err != nil {
		return "", true, err
	}
	return string(data), true, nil
}

func dockerComposeProjectName(info container.InspectResponse) string {
	if info.Config == nil || info.Config.Labels == nil {
		return ""
	}
	return info.Config.Labels[compose.ComposeProjectLabel]
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
