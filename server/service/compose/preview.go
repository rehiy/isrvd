package compose

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/compose-spec/compose-go/v2/types"

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

// PreviewRedeploy 分析部署计划，不写入文件、拉取镜像或修改运行态。
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
	_, after, err := s.resolveRedeployContent(ctx, name, installDir, current.Content, nil, req)
	if err != nil {
		return nil, err
	}
	before, err := compose.LoadProjectFromContentInDir(ctx, current.Content, installDir, name, &envState.Content)
	if err != nil {
		return nil, fmt.Errorf("读取原配置失败: %w", err)
	}
	changes, err := redeployServiceChanges(target, before, after, req.ForcePull)
	if err != nil {
		return nil, err
	}
	return &RedeployPreview{Target: target, ProjectName: name, Changes: changes}, nil
}

// ─── 辅助函数 ───

func redeployServiceChanges(target string, before, after *types.Project, forcePull bool) ([]RedeployServiceChange, error) {
	changes := make([]RedeployServiceChange, 0, len(after.Services))
	for serviceName, newService := range after.Services {
		oldService, existed := before.Services[serviceName]
		change := RedeployServiceChange{Service: serviceName, Action: "create", ConfigChanged: true}
		if target == "swarm" {
			newSpec, err := compose.ServiceToSwarmSpec(after, newService)
			if err != nil {
				return nil, err
			}
			if existed {
				change.ConfigChanged = !swarmServiceSpecMatches(before, serviceName, newSpec)
				change.Action = "update"
				if !change.ConfigChanged && !forcePull {
					change.Action = "unchanged"
				}
			}
		} else {
			newSpec, err := compose.ServiceToDockerCreateSpec(after, newService)
			if err != nil {
				return nil, err
			}
			if existed {
				oldSpec, err := compose.ServiceToDockerCreateSpec(before, oldService)
				change.ConfigChanged = err != nil || !reflect.DeepEqual(oldSpec, newSpec)
				change.Action = "recreate"
			}
		}
		changes = append(changes, change)
	}
	for serviceName := range before.Services {
		if _, ok := after.Services[serviceName]; !ok {
			changes = append(changes, RedeployServiceChange{Service: serviceName, Action: "remove", ConfigChanged: true})
		}
	}
	slices.SortFunc(changes, func(a, b RedeployServiceChange) int {
		return cmp.Compare(a.Service, b.Service)
	})
	return changes, nil
}
