package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			logman.Warn("Restore compose env after failed deploy", "name", projectName, "error", err)
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
		cname := compose.DockerContainerNameOf(svc)
		if _, err := s.docker.ContainerInspect(ctx, cname); err == nil {
			return nil, fmt.Errorf("容器 %s 已存在，请使用重部署接口", cname)
		}
	}

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

func (s *Service) DockerContent(ctx context.Context, name string) (string, string, error) {
	result, err := s.DockerContentResult(ctx, name, false)
	if err != nil {
		return "", "", err
	}
	return result.Content, result.ProjectName, nil
}

// DockerContentResult 读取项目 compose.yml；forceRuntime 为 true 时跳过落盘文件，直接从运行态反推。
func (s *Service) DockerContentResult(ctx context.Context, name string, forceRuntime bool) (*ContentResult, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	projectName := s.dockerProjectName(ctx, name, root)
	path := filepath.Join(root, projectName, "compose.yml")
	fileModTime := composeFileModTime(path)
	if !forceRuntime {
		if data, err := os.ReadFile(path); err == nil {
			envContent, err := compose.EnvContentRead(filepath.Join(root, projectName))
			if err != nil {
				return nil, err
			}
			return &ContentResult{
				Content:     string(data),
				EnvContent:  envContent,
				ProjectName: projectName,
				FileModTime: fileModTime,
				Source:      "file",
			}, nil
		}
	}

	if content, ok, err := s.dockerProjectContentFromContainers(ctx, projectName, root); ok || err != nil {
		if err != nil {
			return nil, err
		}
		envContent, err := compose.EnvContentRead(filepath.Join(root, projectName))
		if err != nil {
			return nil, err
		}
		return &ContentResult{
			Content:     content,
			EnvContent:  envContent,
			ProjectName: projectName,
			FileModTime: fileModTime,
			Source:      "runtime",
		}, nil
	}

	info, err := s.docker.ContainerInspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
	}
	content, err := compose.DockerProjectYAMLFromInspect(ctx, info, s.docker.ImageConfig, filepath.Join(root, name))
	if err != nil {
		return nil, err
	}
	envContent, err := compose.EnvContentRead(filepath.Join(root, projectName))
	if err != nil {
		return nil, err
	}
	return &ContentResult{
		Content:     content,
		EnvContent:  envContent,
		ProjectName: projectName,
		FileModTime: fileModTime,
		Source:      "runtime",
	}, nil
}

// DockerRedeploy 重建 Docker Compose 项目。
// 部分更新：提交哪个字段就改哪个字段，未提交的字段保持不变。
// - ServiceName+Image：仅更新指定服务镜像后重建
// - Content：替换 compose.yml 后重建
// - EnvContent：替换 .env（空串即清空）后重建
func (s *Service) DockerRedeploy(ctx context.Context, name string, req RedeployRequest) (*DeployResult, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
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

	oldEnvState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}

	oldContent, _, contentErr := s.DockerContent(ctx, name)
	content, err := s.prepareRedeployContent(ctx, name, installDir, oldContent, contentErr, req)
	if err != nil {
		return nil, err
	}

	s.dockerContainersRemove(ctx, name, oldContent, installDir, oldEnvState.Content)

	rollback := func() string {
		// 先恢复旧 .env 再回滚容器；.env 失败不阻断容器回滚
		compose.ContentSave(installDir, oldContent, "")
		envErr := compose.EnvStateRestore(installDir, oldEnvState)
		if envErr != nil {
			logman.Warn("Restore compose env before rollback failed", "name", name, "error", envErr)
		}
		runtimeErr := s.dockerRollback(ctx, name, oldContent, installDir)
		if runtimeErr != nil {
			logman.Warn("Rollback containers failed", "name", name, "error", runtimeErr)
		}
		return formatRedeployRollbackSummary(envErr, runtimeErr, "容器")
	}

	// 先落盘新 .env，确保 ProjectLoad 插值读取到新值（与 Deploy 流程顺序一致）
	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	project, err := compose.ProjectLoad(ctx, name, content, installDir)
	if err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	items, err := s.dockerServicesCreate(ctx, project)
	if err != nil {
		return nil, wrapRedeployError(err, rollback())
	}

	compose.ContentSave(installDir, content, oldContent)

	logman.Info("Compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// ==================== 辅助函数 ====================

// dockerServicesCreate 批量创建 project 中的所有容器，失败时回滚。
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

// dockerServiceCreate 根据 compose service 创建对应 Docker 容器（不负责镜像拉取）。
func (s *Service) dockerServiceCreate(ctx context.Context, project *types.Project, svc types.ServiceConfig) (string, string, error) {
	spec, err := compose.ServiceToDockerCreateSpec(project, svc)
	if err != nil {
		return "", "", err
	}
	id, err := s.docker.ContainerCreateAndStart(ctx, spec.Name, spec.Config, spec.HostConfig, spec.NetworkingConfig)
	if err != nil {
		return "", "", fmt.Errorf("创建容器 %s 失败: %w", spec.Name, err)
	}
	return id, spec.Name, nil
}

// dockerContainersRemove 移除 project 中的所有 Docker 容器。
// installDir 作为 compose 加载的工作目录（解析 env_file 等相对路径），envContent 用于叠加插值环境。
func (s *Service) dockerContainersRemove(ctx context.Context, name, content, installDir, envContent string) {
	removed := map[string]struct{}{}
	removeByID := func(id string) {
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

	// 优先通过标签精确查找（ID 级别，无误删风险）
	labelIDs := map[string]struct{}{}
	if infos, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, name); err == nil {
		for _, info := range infos {
			labelIDs[info.ID] = struct{}{}
			removeByID(info.ID)
		}
	} else {
		logman.Warn("List compose project containers failed", "name", name, "error", err)
	}

	// 补充删除无标签的旧容器，inspect 确认归属后再删
	if content != "" {
		project, err := compose.LoadProjectFromContentInDir(ctx, content, installDir, name, &envContent)
		if err != nil {
			// 旧配置解析失败只影响无标签容器的补充清理，标签路径已覆盖大部分场景，
			// 因此不阻断重建，但必须留痕，避免残留容器被静默忽略
			logman.Warn("Parse old compose content for container cleanup failed",
				"name", name, "install_dir", installDir, "error", err)
		} else {
			for _, svc := range project.Services {
				for _, cname := range compose.DockerContainerNameCandidates(name, svc) {
					info, err := s.docker.ContainerInspect(ctx, cname)
					if err != nil {
						continue
					}
					if _, ok := labelIDs[info.ID]; ok {
						continue
					}
					// 归属其他项目，拒绝删除
					if p := compose.DockerComposeProjectName(info); p != "" && p != name {
						logman.Warn("Skip removing container belonging to another project",
							"container", cname, "container_project", p, "expected_project", name)
						continue
					}
					removeByID(info.ID)
				}
			}
		}
	}
}

// dockerProjectName 将容器名/项目名解析为真实 project 名。
// 优先查文件和标签；两者均未命中时，把 name 当容器名 inspect 并读取其 project 标签。
func (s *Service) dockerProjectName(ctx context.Context, name, root string) string {
	if root != "" {
		if _, err := os.Stat(filepath.Join(root, name, "compose.yml")); err == nil {
			return name
		}
	}
	if infos, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, name); err == nil && len(infos) > 0 {
		return name
	}
	if info, err := s.docker.ContainerInspect(ctx, name); err == nil {
		containerName := strings.TrimPrefix(info.Name, "/")
		if containerName == name {
			if projectName := compose.DockerComposeProjectName(info); projectName != "" {
				if err := compose.ValidateProjectName(projectName); err == nil {
					return projectName
				}
				logman.Warn("Ignore invalid compose project label", "container", name, "project", projectName)
			}
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

	content, err := compose.DockerProjectYAMLFromInspects(ctx, infos, s.docker.ImageConfig, projectName, filepath.Join(root, projectName))
	if err != nil {
		return "", true, err
	}
	return content, true, nil
}

// dockerRollback 用指定配置内容重建容器（回滚用）
func (s *Service) dockerRollback(ctx context.Context, name, content, installDir string) error {
	if content == "" {
		return fmt.Errorf("无可回滚的 compose 内容")
	}
	project, err := compose.ProjectParse(ctx, name, content, installDir)
	if err != nil {
		return fmt.Errorf("加载回滚配置失败: %w", err)
	}
	if _, err := s.dockerServicesCreate(ctx, project); err != nil {
		return fmt.Errorf("重建容器失败: %w", err)
	}
	return nil
}

// dockerEnsureNetworks 确保 project 所需网络存在，不存在则创建 bridge 网络
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

func composeFileModTime(path string) int64 {
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return st.ModTime().Unix()
	}
	return 0
}
