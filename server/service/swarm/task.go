package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// Task Swarm 任务信息，保持前端稳定响应结构。
type Task struct {
	ID          string `json:"id"`          // 任务 ID
	ServiceID   string `json:"serviceID"`   // 所属服务 ID
	ServiceName string `json:"serviceName"` // 所属服务名称
	NodeID      string `json:"nodeID"`      // 运行节点 ID
	NodeName    string `json:"nodeName"`    // 运行节点名称
	Slot        int    `json:"slot"`        // 任务槽位
	Image       string `json:"image"`       // 镜像名称
	State       string `json:"state"`       // 任务状态
	Message     string `json:"message"`     // 状态消息
	Err         string `json:"err"`         // 错误信息
	UpdatedAt   string `json:"updatedAt"`   // 更新时间
}

// TaskList 获取任务列表
func (s *Service) TaskList(ctx context.Context, serviceID string) ([]Task, error) {
	tasks, err := s.svc.TaskList(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}
	services, _ := s.svc.ServiceList(ctx)
	nodes, _ := s.svc.NodeList(ctx)
	return tasksFromRaw(tasks, services, nodes), nil
}

func tasksFromRaw(tasks []swarm.Task, services []swarm.Service, nodes []swarm.Node) []Task {
	svcNameMap := map[string]string{}
	for _, svc := range services {
		svcNameMap[svc.ID] = svc.Spec.Name
	}
	nodeNameMap := map[string]string{}
	for _, node := range nodes {
		nodeNameMap[node.ID] = node.Description.Hostname
	}
	result := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		image := ""
		if task.Spec.ContainerSpec != nil {
			image = task.Spec.ContainerSpec.Image
		}
		result = append(result, Task{
			ID: task.ID, ServiceID: task.ServiceID, ServiceName: svcNameMap[task.ServiceID],
			NodeID: task.NodeID, NodeName: nodeNameMap[task.NodeID], Slot: task.Slot, Image: image,
			State: string(task.Status.State), Message: task.Status.Message, Err: task.Status.Err, UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result
}
