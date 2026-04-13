package docker

import (
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// SwarmInfo 获取 Swarm 集群概览
func (h *DockerHandler) SwarmInfo(c *gin.Context) {
	ctx := c.Request.Context()

	info, err := h.dockerClient.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取 Swarm 信息失败: "+err.Error())
		return
	}

	nodes, _ := h.dockerClient.NodeList(ctx, types.NodeListOptions{})
	services, _ := h.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	tasks, _ := h.dockerClient.TaskList(ctx, types.TaskListOptions{})

	var managers, workers int
	for _, n := range nodes {
		if n.Spec.Role == swarm.NodeRoleManager {
			managers++
		} else {
			workers++
		}
	}

	helper.RespondSuccess(c, "Swarm info retrieved", gin.H{
		"clusterID": info.ID,
		"createdAt": info.Meta.CreatedAt.Format(time.RFC3339),
		"nodes":     len(nodes),
		"managers":  managers,
		"workers":   workers,
		"services":  len(services),
		"tasks":     len(tasks),
	})
}

// SwarmListNodes 获取节点列表
func (h *DockerHandler) SwarmListNodes(c *gin.Context) {
	ctx := c.Request.Context()
	nodes, err := h.dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logman.Error("NodeList failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取节点列表失败")
		return
	}

	var result []model.SwarmNode
	for _, n := range nodes {
		result = append(result, model.SwarmNode{
			ID:            n.ID,
			Hostname:      n.Description.Hostname,
			Role:          string(n.Spec.Role),
			Availability:  string(n.Spec.Availability),
			State:         string(n.Status.State),
			Addr:          n.Status.Addr,
			EngineVersion: n.Description.Engine.EngineVersion,
			Leader:        n.ManagerStatus != nil && n.ManagerStatus.Leader,
		})
	}

	helper.RespondSuccess(c, "Nodes listed", result)
}

// SwarmNodeAction 节点操作（drain/active/pause/remove）
func (h *DockerHandler) SwarmNodeAction(c *gin.Context) {
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := c.Request.Context()

	if req.Action == "remove" {
		if err := h.dockerClient.NodeRemove(ctx, req.ID, types.NodeRemoveOptions{Force: true}); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "节点删除失败: "+err.Error())
			return
		}
		helper.RespondSuccess(c, "Node removed", nil)
		return
	}

	node, _, err := h.dockerClient.NodeInspectWithRaw(ctx, req.ID)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "节点不存在")
		return
	}

	switch req.Action {
	case "drain":
		node.Spec.Availability = swarm.NodeAvailabilityDrain
	case "active":
		node.Spec.Availability = swarm.NodeAvailabilityActive
	case "pause":
		node.Spec.Availability = swarm.NodeAvailabilityPause
	default:
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}

	if err := h.dockerClient.NodeUpdate(ctx, req.ID, node.Version, node.Spec); err != nil {
		logman.Error("NodeUpdate failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "节点操作失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Node updated", nil)
}

// SwarmListServices 获取服务列表
func (h *DockerHandler) SwarmListServices(c *gin.Context) {
	ctx := c.Request.Context()
	services, err := h.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		logman.Error("ServiceList failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取服务列表失败")
		return
	}

	// 统计各服务运行中的任务数
	tasks, _ := h.dockerClient.TaskList(ctx, types.TaskListOptions{})
	runningMap := map[string]int{}
	for _, t := range tasks {
		if t.Status.State == swarm.TaskStateRunning {
			runningMap[t.ServiceID]++
		}
	}

	var result []model.SwarmService
	for _, s := range services {
		svc := model.SwarmService{
			ID:           s.ID,
			Name:         s.Spec.Name,
			Image:        s.Spec.TaskTemplate.ContainerSpec.Image,
			Mode:         "replicated",
			RunningTasks: runningMap[s.ID],
			CreatedAt:    s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
		}
		if s.Spec.Mode.Global != nil {
			svc.Mode = "global"
		} else if s.Spec.Mode.Replicated != nil {
			svc.Replicas = s.Spec.Mode.Replicated.Replicas
		}

		var ports []gin.H
		for _, p := range s.Endpoint.Ports {
			if p.PublishedPort > 0 {
				ports = append(ports, gin.H{
					"published": p.PublishedPort,
					"target":    p.TargetPort,
					"protocol":  string(p.Protocol),
				})
			}
		}
		svc.Ports = ports

		result = append(result, svc)
	}

	helper.RespondSuccess(c, "Services listed", result)
}

// SwarmServiceAction 服务操作（scale/remove）
func (h *DockerHandler) SwarmServiceAction(c *gin.Context) {
	var req struct {
		ID       string  `json:"id"`
		Action   string  `json:"action"`
		Replicas *uint64 `json:"replicas,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := c.Request.Context()

	if req.Action == "remove" {
		if err := h.dockerClient.ServiceRemove(ctx, req.ID); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "服务删除失败: "+err.Error())
			return
		}
		helper.RespondSuccess(c, "Service removed", nil)
		return
	}

	if req.Action == "scale" && req.Replicas != nil {
		svc, _, err := h.dockerClient.ServiceInspectWithRaw(ctx, req.ID, types.ServiceInspectOptions{InsertDefaults: true})
		if err != nil {
			helper.RespondError(c, http.StatusNotFound, "服务不存在")
			return
		}
		if svc.Spec.Mode.Replicated == nil {
			helper.RespondError(c, http.StatusBadRequest, "仅 replicated 模式服务支持 scale")
			return
		}
		svc.Spec.Mode.Replicated.Replicas = req.Replicas
		if _, err := h.dockerClient.ServiceUpdate(ctx, req.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{}); err != nil {
			helper.RespondError(c, http.StatusInternalServerError, "服务扩缩容失败: "+err.Error())
			return
		}
		helper.RespondSuccess(c, "Service scaled", nil)
		return
	}

	helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
}

// SwarmListTasks 获取任务列表
func (h *DockerHandler) SwarmListTasks(c *gin.Context) {
	serviceID := c.Query("serviceID")
	ctx := c.Request.Context()

	opts := types.TaskListOptions{}
	if serviceID != "" {
		f := filters.NewArgs()
		f.Add("service", serviceID)
		opts.Filters = f
	}

	tasks, err := h.dockerClient.TaskList(ctx, opts)
	if err != nil {
		logman.Error("TaskList failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取任务列表失败")
		return
	}

	// 建立服务 ID→名称 映射
	services, _ := h.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	svcNameMap := map[string]string{}
	for _, s := range services {
		svcNameMap[s.ID] = s.Spec.Name
	}

	var result []model.SwarmTask
	for _, t := range tasks {
		result = append(result, model.SwarmTask{
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

	helper.RespondSuccess(c, "Tasks listed", result)
}
