package compose

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

// swarmServiceRemoveTimeout 等待旧服务从集群状态中彻底消失的超时时间
const swarmServiceRemoveTimeout = 10 * time.Second

// SwarmDeploy 部署新的 Swarm Compose 项目。
func (s *Service) SwarmDeploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	if s.swarm == nil {
		return nil, fmt.Errorf("Swarm 不可用")
	}
	if err := s.beginDeployment(ctx); err != nil {
		return nil, err
	}
	defer s.deploymentMu.Unlock()
	result, err := s.projectDeploy(ctx, req, "Restore swarm compose env after failed deploy", func(project *types.Project) ([]string, error) {
		for _, svc := range project.Services {
			if _, err := compose.ServiceToSwarmSpec(project, svc); err != nil {
				return nil, err
			}
			if _, err := s.swarm.ServiceInspectRaw(ctx, svc.Name); err == nil {
				return nil, fmt.Errorf("服务 %s 已存在，请先移除", svc.Name)
			} else if !client.IsErrNotFound(err) {
				return nil, err
			}
		}
		return s.swarmServicesCreate(ctx, project)
	})
	if err == nil {
		logman.Info("Swarm compose deployed", "name", result.ProjectName, "dir", result.InstallDir)
		s.historyFinish(ctx, "swarm", result.ProjectName, "deploy", true)
	}
	return result, err
}

// SwarmInspect 读取项目 Compose 配置；forceRuntime 为 true 时跳过落盘文件，直接从运行态反推。
func (s *Service) SwarmInspect(ctx context.Context, name string, forceRuntime bool) (*ConfigDetail, error) {
	if s.swarm == nil {
		return nil, fmt.Errorf("Swarm 不可用")
	}
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	projectName, err := s.swarmProjectName(ctx, name, root)
	if err != nil {
		return nil, err
	}
	installDir := filepath.Join(root, projectName)
	var standaloneID string
	detail, err := inspectComposeConfig(installDir, projectName, forceRuntime, func() (string, error) {
		raw, err := s.swarm.ServiceInspectRaw(ctx, name)
		if err != nil && !client.IsErrNotFound(err) {
			return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
		}
		services := []swarm.Service{}
		if err == nil && (raw.Spec.Labels[compose.ComposeProjectLabel] == "" || raw.Spec.Labels[compose.ComposeProjectLabel] == projectName) {
			services = append(services, raw)
			if raw.Spec.Labels[compose.ComposeProjectLabel] == "" {
				standaloneID = raw.ID
			}
		}
		if err != nil || (len(services) > 0 && raw.Spec.Labels[compose.ComposeProjectLabel] == projectName) || (err == nil && len(services) == 0) {
			services, err = s.swarm.ServiceListByLabel(ctx, compose.ComposeProjectLabel, projectName)
			if err != nil {
				return "", err
			}
		}
		if len(services) == 0 {
			return "", fmt.Errorf("compose 文件不存在且项目 %s 没有可读取的服务", projectName)
		}
		project := &types.Project{Name: projectName, Services: types.Services{}, Networks: types.Networks{}, Volumes: types.Volumes{}}
		for _, item := range services {
			current, err := s.swarm.ServiceInspectRaw(ctx, item.ID)
			if err != nil {
				return "", err
			}
			part, err := compose.ProjectFromSwarmInspect(current, installDir)
			if err != nil {
				return "", err
			}
			maps.Copy(project.Services, part.Services)
			maps.Copy(project.Networks, part.Networks)
			maps.Copy(project.Volumes, part.Volumes)
		}
		data, err := compose.ProjectToYAML(project)
		if err != nil {
			return "", err
		}
		return withGeneratedHeader(string(data)), nil
	})
	if err == nil {
		detail.standaloneID = standaloneID
	}
	return detail, err
}

// SwarmRedeploy 更新 Swarm Compose 项目。
// 部分更新：提交哪个字段就改哪个字段，未提交的字段保持不变。
// - ServiceName+Image：更新指定服务的镜像后协调项目服务
// - Content：替换 compose.yml 后协调项目服务
// - EnvContent：替换 .env（空串即清空）后协调项目服务
func (s *Service) SwarmRedeploy(ctx context.Context, name string, req RedeployRequest) (result *DeployResult, resultErr error) {
	if s.swarm == nil {
		return nil, fmt.Errorf("Swarm 不可用")
	}
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

	root := s.docker.ContainerRoot()
	installDir := ""
	if root != "" {
		projectName, err := s.swarmProjectName(ctx, name, root)
		if err != nil {
			return nil, err
		}
		name = projectName
		installDir = filepath.Join(root, name)
	}

	currentConfig, err := s.SwarmInspect(ctx, name, false)
	if err != nil {
		return nil, err
	}
	oldContent, oldEnv := currentConfig.Content, currentConfig.EnvContent
	content, project, err := s.resolveRedeployContent(ctx, name, installDir, oldContent, req)
	if err != nil {
		return nil, err
	}
	if err := s.historyBegin("swarm", name, currentConfig); err != nil {
		return nil, err
	}
	defer func() { s.historyFinish(ctx, "swarm", name, "redeploy", resultErr == nil) }()

	oldProject, err := compose.LoadProjectFromContentInDir(ctx, oldContent, installDir, name, &oldEnv)
	if err != nil {
		return nil, err
	}
	oldServices := make(map[string]swarm.Service, len(oldProject.Services))
	for _, service := range oldProject.Services {
		current, inspectErr := s.swarm.ServiceInspectRaw(ctx, service.Name)
		if inspectErr != nil {
			if client.IsErrNotFound(inspectErr) {
				continue
			}
			return nil, inspectErr
		}
		if current.Spec.Labels[compose.ComposeProjectLabel] != name && (currentConfig.standaloneID == "" || current.ID != currentConfig.standaloneID) {
			return nil, fmt.Errorf("无法确认服务 %s 属于项目 %s", service.Name, name)
		}
		oldServices[service.Name] = current
	}

	rollback := func(err error) error {
		fileErr, envErr := currentConfig.restore(installDir)
		return wrapRedeployError(ctx, err,
			rollbackResult(ctx, compose.ComposeFileName, fileErr),
			rollbackResult(ctx, compose.EnvFileName, envErr),
		)
	}
	if err := compose.EnvApply(installDir, req.EnvContent); err != nil {
		return nil, rollback(err)
	}
	if err := compose.ProjectSave(content, installDir); err != nil {
		return nil, rollback(err)
	}
	items, err := s.swarmServicesReconcile(ctx, project, oldProject, oldServices, req.ForcePull)
	if err != nil {
		return nil, rollback(err)
	}

	logman.Info("Swarm compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// swarmProjectName 允许以服务名或项目名读取原始项目与部署历史。
func (s *Service) swarmProjectName(ctx context.Context, name, root string) (string, error) {
	if s.swarm == nil {
		return "", fmt.Errorf("Swarm 不可用")
	}
	if root != "" {
		if _, err := os.Stat(filepath.Join(root, name, compose.ComposeFileName)); err == nil {
			return name, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	service, err := s.swarm.ServiceInspectRaw(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return name, nil
		}
		return "", err
	}
	if project := service.Spec.Labels[compose.ComposeProjectLabel]; project != "" {
		if err := compose.ValidateProjectName(project); err != nil {
			return "", err
		}
		return project, nil
	}
	return name, nil
}

// ─── 内部方法 ───

// swarmServicesCreate 批量创建 project 中的所有 Swarm 服务，失败时回滚已创建的服务。
func (s *Service) swarmServicesCreate(ctx context.Context, project *types.Project) (_ []string, resultErr error) {
	networks, err := s.ensureNetworks(ctx, project, "overlay")
	defer s.rollbackNetworks(ctx, networks, &resultErr)
	if err != nil {
		return nil, err
	}

	var createdIDs []string
	var items []string

	rollback := func() error {
		cleanupCtx, cancel := cleanupContext(ctx)
		defer cancel()
		var errs []error
		for _, id := range createdIDs {
			if err := s.swarm.ServiceRemoveAndWait(cleanupCtx, id, swarmServiceRemoveTimeout); err != nil {
				logman.Warn("Rollback remove service failed", "id", id, "error", err)
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	for _, svc := range project.Services {
		id, name, err := s.swarmServiceCreate(ctx, project, svc)
		if err != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				return nil, fmt.Errorf("%w; 服务状态可能未知，请检查运行态; 清理已创建服务失败: %v", err, rollbackErr)
			}
			return nil, fmt.Errorf("%w; 服务状态可能未知，请检查运行态", err)
		}
		createdIDs = append(createdIDs, id)
		items = append(items, fmt.Sprintf("%s (%s)", name, docker.ShortID(id)))
		logman.Info("Swarm service deployed", "service", svc.Name, "id", docker.ShortID(id))
	}
	return items, nil
}

// swarmServiceCreate 根据 compose service 创建对应 Swarm 服务。
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

// swarmServicesReconcile 原地更新已有服务，按旧项目范围删除移除的服务。
func (s *Service) swarmServicesReconcile(ctx context.Context, project, original *types.Project, old map[string]swarm.Service, forcePull bool) (_ []string, resultErr error) {
	names := make([]string, 0, len(project.Services))
	specs := make(map[string]swarm.ServiceSpec, len(project.Services))
	unchanged := make(map[string]bool, len(project.Services))
	for _, service := range project.Services {
		spec, err := compose.ServiceToSwarmSpec(project, service)
		if err != nil {
			return nil, err
		}
		if _, ok := old[spec.Name]; !ok {
			_, err := s.swarm.ServiceInspect(ctx, spec.Name)
			if err == nil {
				return nil, fmt.Errorf("服务 %s 已存在，请先移除", spec.Name)
			}
			if !client.IsErrNotFound(err) {
				return nil, err
			}
		}
		names = append(names, spec.Name)
		specs[spec.Name] = spec
		unchanged[spec.Name] = !forcePull && swarmServiceSpecMatches(original, spec.Name, spec)
	}
	sort.Strings(names)
	networks, err := s.ensureNetworks(ctx, project, "overlay")
	defer s.rollbackNetworks(ctx, networks, &resultErr)
	if err != nil {
		return nil, err
	}
	var created []string
	type serviceUpdate struct{ before, after swarm.Service }
	var updated []serviceUpdate
	var removed []swarm.Service
	rollback := func() error {
		cleanupCtx, cancel := cleanupContext(ctx)
		defer cancel()
		var errs []error
		for i := len(removed) - 1; i >= 0; i-- {
			if _, err := s.swarm.ServiceCreate(cleanupCtx, removed[i].Spec); err != nil {
				errs = append(errs, err)
			}
		}
		for i := len(created) - 1; i >= 0; i-- {
			if err := s.swarm.ServiceRemoveAndWait(cleanupCtx, created[i], swarmServiceRemoveTimeout); err != nil {
				errs = append(errs, err)
			}
		}
		for i := len(updated) - 1; i >= 0; i-- {
			service := updated[i]
			if err := s.swarm.ServiceUpdate(cleanupCtx, service.after, service.before.Spec, false, swarm.RegistryAuthFromPreviousSpec); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	withRollback := func(cause error) error {
		if rollbackErr := rollback(); rollbackErr != nil {
			return fmt.Errorf("%w; 已执行服务回滚，但部分回滚失败: %v", cause, rollbackErr)
		}
		return fmt.Errorf("%w; 先前已确认的服务变更已回滚", cause)
	}
	var items []string
	for _, name := range names {
		spec := specs[name]
		if previous, ok := old[name]; ok && unchanged[name] {
			items = append(items, fmt.Sprintf("%s (%s)", name, docker.ShortID(previous.ID)))
			continue
		}
		id := ""
		if previous, ok := old[name]; ok {
			id = previous.ID
			var tokenBytes [16]byte
			if _, err := rand.Read(tokenBytes[:]); err != nil {
				return nil, withRollback(fmt.Errorf("生成更新标识失败: %w", err))
			}
			token := hex.EncodeToString(tokenBytes[:])
			if spec.Labels == nil {
				spec.Labels = map[string]string{}
			}
			spec.Labels[compose.SwarmUpdateTokenLabel] = token
			if err := s.swarm.ServiceUpdate(ctx, previous, spec, forcePull, swarm.RegistryAuthFromSpec); err != nil {
				cleanupCtx, cancel := cleanupContext(ctx)
				observed, inspectErr := s.swarm.ServiceInspectRaw(cleanupCtx, id)
				cancel()
				status := ""
				if inspectErr == nil && swarmUpdateConfirmed(previous.Spec, observed, token) {
					updated = append(updated, serviceUpdate{before: previous, after: observed})
				} else if inspectErr != nil || observed.Version.Index != previous.Version.Index {
					status = "; 此服务状态未知，请检查运行态"
				}
				return nil, withRollback(fmt.Errorf("更新服务 %s 失败: %w%s", name, err, status))
			}
			after, err := s.swarm.ServiceInspectRaw(ctx, id)
			if err != nil {
				checkCtx, cancel := cleanupContext(ctx)
				observed, checkErr := s.swarm.ServiceInspectRaw(checkCtx, id)
				cancel()
				if checkErr == nil && swarmUpdateConfirmed(previous.Spec, observed, token) {
					updated = append(updated, serviceUpdate{before: previous, after: observed})
					return nil, withRollback(fmt.Errorf("服务 %s 更新后首次读取失败: %w", name, err))
				}
				return nil, withRollback(fmt.Errorf("服务 %s 更新后读取失败: %w; 此服务状态未知，请检查运行态", name, err))
			}
			if !swarmUpdateConfirmed(previous.Spec, after, token) {
				return nil, withRollback(fmt.Errorf("服务 %s 更新后状态已变化，未回滚该服务，请检查运行态", name))
			}
			updated = append(updated, serviceUpdate{before: previous, after: after})
		} else {
			var err error
			id, err = s.swarm.ServiceCreate(ctx, spec)
			if err != nil {
				return nil, withRollback(fmt.Errorf("创建服务 %s 失败: %w; 服务状态可能未知，请检查运行态", name, err))
			}
			created = append(created, id)
		}
		items = append(items, fmt.Sprintf("%s (%s)", name, docker.ShortID(id)))
	}
	removedNames := make([]string, 0, len(old))
	for name := range old {
		removedNames = append(removedNames, name)
	}
	sort.Strings(removedNames)
	for _, name := range removedNames {
		previous := old[name]
		if _, ok := specs[name]; ok {
			continue
		}
		current, err := s.swarm.ServiceInspectRaw(ctx, previous.ID)
		if err != nil {
			return nil, withRollback(fmt.Errorf("移除服务 %s 前读取失败: %w", name, err))
		}
		if current.Version.Index != previous.Version.Index {
			return nil, withRollback(fmt.Errorf("移除服务 %s 前状态已变化", name))
		}
		if err := s.swarm.ServiceRemoveAndWait(ctx, previous.ID, swarmServiceRemoveTimeout); err != nil {
			cleanupCtx, cancel := cleanupContext(ctx)
			_, inspectErr := s.swarm.ServiceInspectRaw(cleanupCtx, previous.ID)
			cancel()
			if client.IsErrNotFound(inspectErr) {
				removed = append(removed, previous)
			}
			return nil, withRollback(fmt.Errorf("移除服务 %s 失败或状态未知: %w", name, err))
		}
		removed = append(removed, previous)
	}
	return items, nil
}

// swarmServiceSpecMatches 比较原项目中的服务规格，供预览和实际重部署使用。
func swarmServiceSpecMatches(original *types.Project, name string, spec swarm.ServiceSpec) bool {
	if original == nil {
		return false
	}
	previous, ok := original.Services[name]
	if !ok {
		return false
	}
	oldSpec, err := compose.ServiceToSwarmSpec(original, previous)
	return err == nil && reflect.DeepEqual(oldSpec, spec)
}

func swarmUpdateConfirmed(before swarm.ServiceSpec, after swarm.Service, token string) bool {
	return after.Spec.Labels[compose.SwarmUpdateTokenLabel] == token && swarmPreviousSpecMatches(before, after.PreviousSpec)
}

// swarmPreviousSpecMatches 仅在 Docker 确认前一个 spec 就是快照时才允许自动回滚。
func swarmPreviousSpecMatches(before swarm.ServiceSpec, previous *swarm.ServiceSpec) bool {
	if previous == nil {
		return false
	}
	a, err := json.Marshal(before)
	if err != nil {
		return false
	}
	b, err := json.Marshal(previous)
	return err == nil && string(a) == string(b)
}
