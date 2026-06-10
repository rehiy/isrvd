package swarm

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/libgo/logman"
)

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
