package compose

import (
	"context"
	"fmt"
	"slices"

	"github.com/docker/docker/api/types/swarm"

	"isrvd/pkgs/compose"
)

// SwarmStackStatus 由 Docker Engine 当前任务状态汇总，客户端可以重复请求观察更新过程。
type SwarmStackStatus struct {
	ProjectName string               `json:"projectName"`
	Services    []SwarmServiceStatus `json:"services"`
}

type SwarmServiceStatus struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	UpdateState   string `json:"updateState,omitempty"` // updating/paused/completed/rollback_*
	UpdateMessage string `json:"updateMessage,omitempty"`
	RunningTasks  int    `json:"runningTasks"`
	DesiredTasks  int    `json:"desiredTasks"`
	Mode          string `json:"mode"` // replicated/global
}

// SwarmStatus 查询当前 Swarm 更新和副本状态，不将 API 调用成功等同于任务健康。
func (s *Service) SwarmStatus(ctx context.Context, name string) (*SwarmStackStatus, error) {
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
	// 与 SwarmInspect 一致：以项目标签筛选，避免误读其他项目服务。
	services, err := s.swarm.ServiceListByLabel(ctx, compose.ComposeProjectLabel, projectName)
	if err != nil {
		return nil, err
	}
	result := &SwarmStackStatus{ProjectName: projectName, Services: []SwarmServiceStatus{}}
	for _, item := range services {
		current, err := s.swarm.ServiceInspectRaw(ctx, item.ID)
		if err != nil {
			return nil, fmt.Errorf("读取服务 %s 状态失败: %w", item.Spec.Name, err)
		}
		// 按服务过滤，避免每次状态刷新扫描集群中所有项目的任务。
		tasks, err := s.swarm.TaskList(ctx, item.ID)
		if err != nil {
			return nil, fmt.Errorf("读取服务 %s 任务失败: %w", item.Spec.Name, err)
		}
		var running, desired int
		for _, task := range tasks {
			if task.DesiredState != swarm.TaskStateRunning {
				continue
			}
			desired++
			if task.Status.State == swarm.TaskStateRunning {
				running++
			}
		}
		detail := SwarmServiceStatus{
			ID: item.ID, Name: current.Spec.Name,
			RunningTasks: running, DesiredTasks: desired, Mode: "global",
		}
		if replicated := current.Spec.Mode.Replicated; replicated != nil {
			detail.Mode = "replicated"
			if replicated.Replicas != nil {
				detail.DesiredTasks = int(*replicated.Replicas)
			}
		}
		if current.UpdateStatus != nil {
			detail.UpdateState = string(current.UpdateStatus.State)
			detail.UpdateMessage = current.UpdateStatus.Message
		}
		result.Services = append(result.Services, detail)
	}
	slices.SortFunc(result.Services, func(a, b SwarmServiceStatus) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result, nil
}
