package swarm

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/libgo/logman"
)

// NodeTaskList 获取指定节点上的任务，保留实际状态供业务层判断。
func (s *SwarmService) NodeTaskList(ctx context.Context, nodeID string) ([]swarm.Task, error) {
	return s.client.TaskList(ctx, swarm.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("node", nodeID)),
	})
}

// TaskList 获取任务列表，直接返回 Docker SDK 原始任务结构。
func (s *SwarmService) TaskList(ctx context.Context, serviceID string) ([]swarm.Task, error) {
	opts := swarm.TaskListOptions{}
	if serviceID != "" {
		f := filters.NewArgs()
		f.Add("service", serviceID)
		opts.Filters = f
	}

	tasks, err := s.client.TaskList(ctx, opts)
	if err != nil {
		logman.Error("TaskList failed", "error", err)
		return nil, err
	}
	return tasks, nil
}
