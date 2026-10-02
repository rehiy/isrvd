package compose

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/compose"
	"isrvd/pkgs/docker"
)

// swarmServiceRemoveTimeout 等待旧服务从集群状态中彻底消失的超时时间
const swarmServiceRemoveTimeout = 10 * time.Second

// SwarmDeploy 部署新的 Swarm Compose 项目。
func (s *Service) SwarmDeploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	if err := s.beginDeployment(ctx); err != nil {
		return nil, err
	}
	defer s.deploymentMu.Unlock()
	result, err := s.projectDeploy(ctx, req, "Restore swarm compose env after failed deploy", func(project *types.Project) ([]string, error) {
		for _, svc := range project.Services {
			if _, err := s.swarm.ServiceInspect(ctx, svc.Name); err == nil {
				return nil, fmt.Errorf("服务 %s 已存在，请先移除", svc.Name)
			}
		}
		if err := s.imagesEnsure(ctx, project, req.ForcePull); err != nil {
			return nil, err
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
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}

	installDir := filepath.Join(root, name)
	return inspectComposeConfig(installDir, name, forceRuntime, func() (string, error) {
		raw, err := s.swarm.ServiceInspect(ctx, name)
		if err != nil {
			return "", fmt.Errorf("compose 文件不存在且读取运行态失败: %w", err)
		}
		project, err := compose.ProjectFromSwarmInspect(raw, installDir)
		if err != nil {
			return "", err
		}
		data, err := compose.ProjectToYAML(project)
		return string(data), err
	})
}

// SwarmRedeploy 重建 Swarm Compose 项目。
// 部分更新：提交哪个字段就改哪个字段，未提交的字段保持不变。
// - ServiceName+Image：仅更新指定服务的镜像后全量重建
// - Content：替换 compose.yml 后重建
// - EnvContent：替换 .env（空串即清空）后重建
func (s *Service) SwarmRedeploy(ctx context.Context, name string, req RedeployRequest) (result *DeployResult, resultErr error) {
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
		installDir = filepath.Join(root, name)
	}

	oldEnvState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}

	currentConfig, contentErr := s.SwarmInspect(ctx, name, false)
	oldContent := ""
	if currentConfig != nil {
		oldContent = currentConfig.Content
	}
	content, err := s.prepareRedeployContent(ctx, name, installDir, oldContent, contentErr, req)
	if err != nil {
		return nil, err
	}
	if err := s.historyBegin("swarm", name, currentConfig); err != nil {
		return nil, err
	}
	defer func() { s.historyFinish(ctx, "swarm", name, "redeploy", resultErr == nil) }()

	// 旧服务尚未确认移除干净前不能进入创建阶段；部分移除失败时尝试恢复旧服务。
	if err := s.swarmServicesRemove(ctx, name, oldContent, installDir, oldEnvState.Content); err != nil {
		cleanupCtx, cleanupCancel := cleanupContext(ctx)
		cleanupErr := s.swarmServicesRemove(cleanupCtx, name, oldContent, installDir, oldEnvState.Content)
		cleanupCancel()
		var rollbackErr error
		if cleanupErr == nil {
			rollbackCtx, rollbackCancel := cleanupContext(ctx)
			rollbackErr = s.swarmRollback(rollbackCtx, name, oldContent, installDir)
			rollbackCancel()
		} else {
			rollbackErr = fmt.Errorf("继续清理旧服务失败: %w", cleanupErr)
		}
		return nil, wrapRedeployError(fmt.Errorf("移除旧服务失败: %w", err), formatRedeployRollbackSummary(nil, rollbackErr, "服务"))
	}

	rollback := func() string {
		// 先恢复旧 .env 再回滚服务；.env 失败不阻断服务回滚
		compose.ContentSave(installDir, oldContent)
		envErr := compose.EnvStateRestore(installDir, oldEnvState)
		if envErr != nil {
			logman.Warn("Restore swarm compose env before rollback failed", "name", name, "error", envErr)
		}
		rollbackCtx, cancel := cleanupContext(ctx)
		defer cancel()
		runtimeErr := s.swarmRollback(rollbackCtx, name, oldContent, installDir)
		if runtimeErr != nil {
			logman.Warn("Rollback swarm services failed", "name", name, "error", runtimeErr)
		}
		return formatRedeployRollbackSummary(envErr, runtimeErr, "服务")
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

	logman.Info("Swarm compose redeployed", "name", name)
	return &DeployResult{ProjectName: name, Items: items, InstallDir: installDir}, nil
}

// ==================== 内部方法 ====================

// swarmServicesCreate 批量创建 project 中的所有 Swarm 服务，失败时回滚已创建的服务。
// 调用前须先通过 imagesEnsure 完成预拉取。
func (s *Service) swarmServicesCreate(ctx context.Context, project *types.Project) ([]string, error) {
	if err := s.swarmEnsureNetworks(ctx, project); err != nil {
		return nil, err
	}

	var createdIDs []string
	var items []string

	rollback := func() {
		cleanupCtx, cancel := cleanupContext(ctx)
		defer cancel()
		for _, id := range createdIDs {
			if err := s.swarm.ServiceRemoveAndWait(cleanupCtx, id, swarmServiceRemoveTimeout); err != nil {
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

// swarmServicesRemove 移除 project 中的所有 Swarm 服务，并等待其从集群状态中彻底消失，
// 避免紧随其后的同名 create 撞上 Docker daemon 异步清理导致的 AlreadyExists。
// installDir 作为 compose 加载的工作目录（解析 env_file 等相对路径），envContent 用于叠加插值环境。
func (s *Service) swarmServicesRemove(ctx context.Context, name, content, installDir, envContent string) error {
	if content == "" {
		return nil
	}
	project, err := compose.LoadProjectFromContentInDir(ctx, content, installDir, name, &envContent)
	if err != nil {
		return err
	}
	var removeErrors []error
	for _, svc := range project.Services {
		if err := s.swarm.ServiceRemoveAndWait(ctx, svc.Name, swarmServiceRemoveTimeout); err != nil {
			removeErrors = append(removeErrors, fmt.Errorf("移除服务 %s 失败: %w", svc.Name, err))
		}
	}
	return errors.Join(removeErrors...)
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
