package compose

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

// DockerDeploy 部署新的 Docker Compose 项目。
func (s *Service) DockerDeploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	if err := s.beginDeployment(ctx); err != nil {
		return nil, err
	}
	defer s.deploymentMu.Unlock()
	result, err := s.projectDeploy(ctx, req, "Restore compose env after failed deploy", func(project *types.Project) (_ []string, resultErr error) {
		specs, err := dockerCreateSpecs(project)
		if err != nil {
			return nil, err
		}
		if err := s.dockerCheckNames(ctx, specs, nil); err != nil {
			return nil, err
		}
		if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
			return nil, err
		}
		networks, err := s.ensureNetworks(ctx, project, "bridge")
		defer s.rollbackNetworks(ctx, networks, &resultErr)
		if err != nil {
			return nil, err
		}
		return s.dockerServicesCreate(ctx, specs)
	})
	if err == nil {
		logman.Info("Compose deployed", "name", result.ProjectName, "dir", result.InstallDir)
		s.historyFinish(ctx, "docker", result.ProjectName, "deploy", true)
	}
	return result, err
}

// DockerInspect 读取项目 Compose 配置；forceRuntime 为 true 时跳过落盘文件，直接从运行态反推。
func (s *Service) DockerInspect(ctx context.Context, name string, forceRuntime bool) (*ConfigDetail, error) {
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	projectName := s.dockerProjectName(ctx, name, root)
	var standaloneID string
	detail, err := inspectComposeConfig(filepath.Join(root, projectName), projectName, forceRuntime, func() (string, error) {
		if content, ok, err := s.dockerProjectContentFromContainers(ctx, projectName, root); ok || err != nil {
			return content, err
		}

		info, err := s.docker.ContainerInspect(ctx, name)
		if err != nil {
			return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
		}
		content, err := compose.DockerProjectYAMLFromInspect(ctx, info, s.docker.ImageConfig, filepath.Join(root, name))
		if err != nil {
			return "", err
		}
		if compose.DockerComposeProjectName(info) == "" {
			standaloneID = info.ID
		}
		return withGeneratedHeader(content), nil
	})
	if err == nil {
		detail.standaloneID = standaloneID
	}
	return detail, err
}

// DockerRedeploy 重建 Docker Compose 项目。
// 部分更新：提交哪个字段就改哪个字段，未提交的字段保持不变。
// - ServiceName+Image：仅更新指定服务镜像后重建
// - Content：替换 compose.yml 后重建
// - EnvContent：替换 .env（空串即清空）后重建
func (s *Service) DockerRedeploy(ctx context.Context, name string, req RedeployRequest) (result *DeployResult, resultErr error) {
	if err := s.beginDeployment(ctx); err != nil {
		return nil, err
	}
	defer s.deploymentMu.Unlock()
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	current, err := s.DockerInspect(ctx, name, false)
	if err != nil {
		return nil, err
	}
	name = current.ProjectName
	installDir := filepath.Join(s.docker.ContainerRoot(), name)
	content, project, err := s.resolveRedeployContent(ctx, name, installDir, current.Content, req)
	if err != nil {
		return nil, err
	}
	specs, err := dockerCreateSpecs(project)
	if err != nil {
		return nil, err
	}
	previous, err := s.dockerPreviousContainers(ctx, current)
	if err != nil {
		return nil, err
	}
	if err := s.dockerCheckNames(ctx, specs, previous); err != nil {
		return nil, err
	}
	if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
		return nil, err
	}
	if err := s.historyBegin("docker", name, current); err != nil {
		return nil, err
	}
	defer func() { s.historyFinish(ctx, "docker", name, "redeploy", resultErr == nil) }()

	networks, err := s.ensureNetworks(ctx, project, "bridge")
	defer s.rollbackNetworks(ctx, networks, &resultErr)
	if err != nil {
		return nil, err
	}

	removed, err := s.dockerRemovePrevious(ctx, previous)
	if err != nil {
		return nil, wrapRedeployError(ctx, err, rollbackResult(ctx, "容器", s.dockerRestore(ctx, removed)))
	}

	rollback := func(err error) error {
		fileErr, envErr := current.restore(installDir)
		runtimeErr := s.dockerRestore(ctx, removed)
		return wrapRedeployError(ctx, err,
			rollbackResult(ctx, compose.ComposeFileName, fileErr),
			rollbackResult(ctx, compose.EnvFileName, envErr),
			rollbackResult(ctx, "容器", runtimeErr),
		)
	}
	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, rollback(err)
	}
	if err := compose.ProjectSave(content, installDir); err != nil {
		return nil, rollback(err)
	}
	items, err := s.dockerServicesCreate(ctx, specs)
	if err != nil {
		return nil, rollback(err)
	}

	logman.Info("Compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

func dockerCreateSpecs(project *types.Project) ([]compose.DockerCreateSpec, error) {
	seen := make(map[string]bool, len(project.Services))
	specs := make([]compose.DockerCreateSpec, 0, len(project.Services))
	for _, svc := range project.Services {
		spec, err := compose.ServiceToDockerCreateSpec(project, svc)
		if err != nil {
			return nil, fmt.Errorf("服务 %s 预检失败: %w", svc.Name, err)
		}
		if seen[spec.Name] {
			return nil, fmt.Errorf("重部署预检失败: 容器名称 %s 重复", spec.Name)
		}
		seen[spec.Name] = true
		specs = append(specs, spec)
	}
	return specs, nil
}

func (s *Service) dockerCheckNames(ctx context.Context, specs []compose.DockerCreateSpec, previous []container.InspectResponse) error {
	owned := make(map[string]bool, len(previous))
	for _, item := range previous {
		owned[strings.TrimPrefix(item.Name, "/")] = true
	}
	for _, spec := range specs {
		if owned[spec.Name] {
			continue
		}
		_, err := s.docker.ContainerInspect(ctx, spec.Name)
		if err == nil {
			return fmt.Errorf("容器名称 %s 已被占用", spec.Name)
		}
		if !client.IsErrNotFound(err) {
			return err
		}
	}
	return nil
}

// dockerPreviousContainers 仅选择有项目标签的容器，以及反推配置时明确选中的单个容器。
func (s *Service) dockerPreviousContainers(ctx context.Context, current *ConfigDetail) ([]container.InspectResponse, error) {
	list, err := s.docker.ContainerListByLabel(ctx, compose.ComposeProjectLabel, current.ProjectName)
	if err != nil {
		return nil, err
	}
	list = slices.DeleteFunc(list, func(info container.InspectResponse) bool {
		return compose.DockerComposeProjectName(info) != current.ProjectName ||
			strings.EqualFold(info.Config.Labels[compose.ComposeOneoffLabel], "true")
	})
	if current.standaloneID != "" {
		info, err := s.docker.ContainerInspect(ctx, current.standaloneID)
		if err != nil {
			return nil, err
		}
		list = append(list, info)
	}
	for _, info := range list {
		if info.Config == nil || info.HostConfig == nil || info.State == nil || info.Image == "" {
			return nil, fmt.Errorf("容器 inspect 数据不完整: %s", info.ID)
		}
	}
	slices.SortFunc(list, func(a, b container.InspectResponse) int { return cmp.Compare(a.Created, b.Created) })
	return list, nil
}

func (s *Service) dockerRemovePrevious(ctx context.Context, previous []container.InspectResponse) ([]container.InspectResponse, error) {
	removed := make([]container.InspectResponse, 0, len(previous))
	for _, item := range previous {
		if err := s.docker.ContainerAction(ctx, item.ID, "remove"); err != nil {
			check, cancel := cleanupContext(ctx)
			_, inspectErr := s.docker.ContainerInspect(check, item.ID)
			cancel()
			if client.IsErrNotFound(inspectErr) {
				removed = append(removed, item)
			} else if inspectErr != nil {
				err = errors.Join(err, inspectErr)
			}
			return removed, fmt.Errorf("移除容器 %s 失败: %w", docker.ShortID(item.ID), err)
		}
		removed = append(removed, item)
	}
	return removed, nil
}

// dockerServicesCreate 按已预检的规格创建与启动容器，失败时清理本批创建的容器。
func (s *Service) dockerServicesCreate(ctx context.Context, specs []compose.DockerCreateSpec) (_ []string, resultErr error) {
	var created, items []string
	defer func() {
		if resultErr == nil || len(created) == 0 {
			return
		}
		cleanup, cancel := cleanupContext(ctx)
		defer cancel()
		for _, id := range slices.Backward(created) {
			if err := s.docker.ContainerAction(cleanup, id, "remove"); err != nil && !client.IsErrNotFound(err) {
				resultErr = errors.Join(resultErr, fmt.Errorf("移除容器 %s 失败: %w", docker.ShortID(id), err))
			}
		}
	}()
	for _, spec := range specs {
		id, err := s.docker.ContainerCreate(ctx, spec.Name, spec.Config, spec.HostConfig, spec.NetworkingConfig)
		if err != nil {
			return nil, fmt.Errorf("创建容器 %s 失败: %w", spec.Name, err)
		}
		created = append(created, id)
		if err := s.docker.ContainerStart(ctx, id); err != nil {
			return nil, err
		}
		items = append(items, fmt.Sprintf("%s (%s)", spec.Name, docker.ShortID(id)))
	}
	return items, nil
}

// dockerRestore 只恢复本次已移除的容器，并使用其镜像 ID 与运行状态。
func (s *Service) dockerRestore(ctx context.Context, removed []container.InspectResponse) error {
	if len(removed) == 0 {
		return nil
	}
	cleanup, cancel := cleanupContext(ctx)
	defer cancel()
	var failures []error
	for _, old := range removed {
		config := *old.Config
		config.Image = old.Image
		name := strings.TrimPrefix(old.Name, "/")
		networks := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
		if old.NetworkSettings != nil {
			for key, endpoint := range old.NetworkSettings.Networks {
				if endpoint != nil {
					networks.EndpointsConfig[key] = &network.EndpointSettings{
						IPAMConfig: endpoint.IPAMConfig, Aliases: endpoint.Aliases, Links: endpoint.Links,
						DriverOpts: endpoint.DriverOpts, GwPriority: endpoint.GwPriority,
					}
				}
			}
		}
		host := restoreDockerVolumes(old)
		id, err := s.docker.ContainerCreate(cleanup, name, &config, &host, networks)
		if err == nil && old.State.Running {
			err = s.docker.ContainerStart(cleanup, id)
			if err == nil && old.State.Paused {
				err = s.docker.ContainerAction(cleanup, id, "pause")
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("恢复容器 %s 失败: %w", name, err))
		}
	}
	return errors.Join(failures...)
}

// restoreDockerVolumes 复用旧容器实际挂载的匿名卷，避免重建时分配空卷。
func restoreDockerVolumes(old container.InspectResponse) container.HostConfig {
	host := *old.HostConfig
	host.Mounts, host.Binds = slices.Clone(host.Mounts), slices.Clone(host.Binds)
	for _, volume := range old.Mounts {
		if volume.Type != mount.TypeVolume || volume.Name == "" {
			continue
		}
		if index := slices.IndexFunc(host.Mounts, func(m mount.Mount) bool { return m.Target == volume.Destination }); index >= 0 {
			host.Mounts[index].Source = volume.Name
			continue
		}
		bind := volume.Name + ":" + volume.Destination
		if index := slices.Index(host.Binds, volume.Destination); index >= 0 {
			host.Binds[index] = bind
		} else if !slices.ContainsFunc(host.Binds, func(value string) bool { return value == bind || strings.HasPrefix(value, bind+":") }) {
			host.Mounts = append(host.Mounts, mount.Mount{Type: mount.TypeVolume, Source: volume.Name, Target: volume.Destination, ReadOnly: !volume.RW})
		}
	}
	return host
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
	return withGeneratedHeader(content), true, nil
}
