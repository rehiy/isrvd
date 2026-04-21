package swarm

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/pango/logman"
)

// Task Swarm 任务信息
type Task struct {
	ID          string `json:"id"`
	ServiceID   string `json:"serviceID"`
	ServiceName string `json:"serviceName"`
	NodeID      string `json:"nodeID"`
	Slot        int    `json:"slot"`
	Image       string `json:"image"`
	State       string `json:"state"`
	Message     string `json:"message"`
	Err         string `json:"err"`
	UpdatedAt   string `json:"updatedAt"`
}

// ListTasks 获取任务列表
func (m *SwarmService) ListTasks(ctx context.Context, serviceID string) ([]Task, error) {
	opts := swarm.TaskListOptions{}
	if serviceID != "" {
		f := filters.NewArgs()
		f.Add("service", serviceID)
		opts.Filters = f
	}

	tasks, err := m.client.TaskList(ctx, opts)
	if err != nil {
		logman.Error("TaskList failed", "error", err)
		return nil, err
	}

	// 建立服务 ID→名称 映射
	services, _ := m.client.ServiceList(ctx, swarm.ServiceListOptions{})
	svcNameMap := map[string]string{}
	for _, s := range services {
		svcNameMap[s.ID] = s.Spec.Name
	}

	var result []Task
	for _, t := range tasks {
		result = append(result, Task{
			ID:          t.ID,
			ServiceID:   t.ServiceID,
			ServiceName: svcNameMap[t.ServiceID],
			NodeID:      t.NodeID,
			Slot:        t.Slot,
			Image:       t.Spec.ContainerSpec.Image,
			State:       string(t.Status.State),
			Message:     t.Status.Message,
			Err:         t.Status.Err,
			UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}
