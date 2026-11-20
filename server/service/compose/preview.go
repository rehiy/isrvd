package compose

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"


	"isrvd/pkgs/compose"
)

// RedeployPreview 只读变更预览；不代表 Docker/Swarm 实际状态或健康结果。
type RedeployPreview struct {
	Target      string                  `json:"target"`
	ProjectName string                  `json:"projectName"`
	Changes     []RedeployServiceChange `json:"changes"`
}

// RedeployServiceChange 描述单个服务的预期操作。
type RedeployServiceChange struct {
	Service       string `json:"service"`
	Action        string `json:"action"` // create / recreate / update / unchanged / remove
	ConfigChanged bool   `json:"configChanged"`
}

// RedeployPreview 在不写磁盘、不拉取镜像、不触碰运行态的前提下分析部署计划。
func (s *Service) PreviewRedeploy(ctx context.Context, target, name string, req RedeployRequest) (*RedeployPreview, error) {
	if target != "docker" && target != "swarm" {
		return nil, fmt.Errorf("不支持的 compose 目标 %s", target)
	}
	if err := compose.ValidateProjectName(name); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.beginDeployment(ctx); err != nil {
		return nil, err
	}
	defer s.deploymentMu.Unlock()

	root := s.docker.ContainerRoot()
	if root == "" {
		return nil, fmt.Errorf("未配置容器数据根目录")
	}
	if target == "swarm" {
		if s.swarm == nil {
			return nil, fmt.Errorf("Swarm 不可用")
		}
		var err error
		name, err = s.swarmProjectName(ctx, name, root)
		if err != nil {
			return nil, err
		}
	} else {
		name = s.dockerProjectName(ctx, name, root)
	}
	installDir := filepath.Join(root, name)
	envState, err := compose.EnvStateRead(installDir)
	if err != nil {
		return nil, err
	}
	var current *ConfigDetail
	if target == "swarm" {
		current, err = s.SwarmInspect(ctx, name, false)
	} else {
		current, err = s.DockerInspect(ctx, name, false)
	}
	if err != nil {
		return nil, err
	}
	content, after, err := s.resolveRedeployContent(ctx, name, installDir, current.Content, nil, req)
	if err != nil {
		return nil, err
	}
	before, err := compose.LoadProjectFromContentInDir(ctx, current.Content, installDir, name, &envState.Content)
	if err != nil {
		return nil, fmt.Errorf("读取原配置失败: %w", err)
	}
	result := &RedeployPreview{Target: target, ProjectName: name, Changes: []RedeployServiceChange{}}
	for serviceName, newService := range after.Services {
		oldService, existed := before.Services[serviceName]
		change := RedeployServiceChange{Service: serviceName, Action: "create", ConfigChanged: true}
		if existed {
			var same bool
			if target == "swarm" {
				oldSpec, oldErr := compose.ServiceToSwarmSpec(before, oldService)
				newSpec, newErr := compose.ServiceToSwarmSpec(after, newService)
				if newErr != nil {
					return nil, newErr
				}
				same = oldErr == nil && reflect.DeepEqual(oldSpec, newSpec)
				change.Action = "update"
				if same && !req.ForcePull {
					change.Action = "unchanged"
				}
			} else {
				oldSpec, oldErr := compose.ServiceToDockerCreateSpec(before, oldService)
				newSpec, newErr := compose.ServiceToDockerCreateSpec(after, newService)
				if newErr != nil {
					return nil, newErr
				}
				same = oldErr == nil && reflect.DeepEqual(oldSpec, newSpec)
				change.Action = "recreate"
			}
			change.ConfigChanged = !same
		} else if target == "swarm" {
			if _, err := compose.ServiceToSwarmSpec(after, newService); err != nil {
				return nil, err
			}
		} else {
			if _, err := compose.ServiceToDockerCreateSpec(after, newService); err != nil {
				return nil, err
			}
		}
		result.Changes = append(result.Changes, change)
	}
	for serviceName := range before.Services {
		if _, ok := after.Services[serviceName]; !ok {
			result.Changes = append(result.Changes, RedeployServiceChange{Service: serviceName, Action: "remove", ConfigChanged: true})
		}
	}
	slices.SortFunc(result.Changes, func(a, b RedeployServiceChange) int {
		if a.Service < b.Service {
			return -1
		}
		if a.Service > b.Service {
			return 1
		}
		return 0
	})
	return result, nil
}
