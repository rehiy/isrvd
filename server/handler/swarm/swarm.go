package swarm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"

)

// SwarmInfo 获取 Swarm 集群概览
func (h *SwarmHandler) SwarmInfo(c *gin.Context) {
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
func (h *SwarmHandler) SwarmListNodes(c *gin.Context) {
	ctx := c.Request.Context()
	nodes, err := h.dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logman.Error("NodeList failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取节点列表失败")
		return
	}

	var result []SwarmNode
	for _, n := range nodes {
		result = append(result, SwarmNode{
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
func (h *SwarmHandler) SwarmNodeAction(c *gin.Context) {
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
func (h *SwarmHandler) SwarmListServices(c *gin.Context) {
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

	var result []SwarmService
	for _, s := range services {
		svc := SwarmService{
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
func (h *SwarmHandler) SwarmServiceAction(c *gin.Context) {
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
func (h *SwarmHandler) SwarmListTasks(c *gin.Context) {
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

	var result []SwarmTask
	for _, t := range tasks {
		result = append(result, SwarmTask{
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

// SwarmCreateService 创建服务
func (h *SwarmHandler) SwarmCreateService(c *gin.Context) {
	var req SwarmCreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := c.Request.Context()

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{Name: req.Name},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: req.Image,
				Env:   req.Env,
				Args:  req.Args,
			},
		},
		EndpointSpec: &swarm.EndpointSpec{},
	}

	// 副本数
	if req.Mode == "global" {
		spec.Mode = swarm.ServiceMode{Global: &swarm.GlobalService{}}
	} else {
		replicas := uint64(req.Replicas)
		if replicas == 0 {
			replicas = 1
		}
		spec.Mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
	}

	// 端口映射
	for _, p := range req.Ports {
		proto := swarm.PortConfigProtocolTCP
		if strings.EqualFold(p.Protocol, "udp") {
			proto = swarm.PortConfigProtocolUDP
		}
		spec.EndpointSpec.Ports = append(spec.EndpointSpec.Ports, swarm.PortConfig{
			Protocol:      proto,
			PublishedPort: uint32(p.Published),
			TargetPort:    uint32(p.Target),
		})
	}

	// 挂载卷
	for _, m := range req.Mounts {
		mt := mount.TypeBind
		if m.Type == "volume" {
			mt = mount.TypeVolume
		}
		spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, mount.Mount{
			Type:   mt,
			Source: m.Source,
			Target: m.Target,
		})
	}

	// 网络
	for _, n := range req.Networks {
		spec.Networks = append(spec.Networks, swarm.NetworkAttachmentConfig{Target: n})
	}

	resp, err := h.dockerClient.ServiceCreate(ctx, spec, types.ServiceCreateOptions{})
	if err != nil {
		logman.Error("ServiceCreate failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "服务创建失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service created", gin.H{"id": resp.ID})
}

// SwarmForceUpdateService 强制重新部署服务（不改配置，仅 force update）
func (h *SwarmHandler) SwarmForceUpdateService(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx := c.Request.Context()
	svc, _, err := h.dockerClient.ServiceInspectWithRaw(ctx, req.ID, types.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "服务不存在")
		return
	}

	// 触发 force update：将 ForceUpdate 字段 +1
	svc.Spec.TaskTemplate.ForceUpdate++

	if _, err := h.dockerClient.ServiceUpdate(ctx, req.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{}); err != nil {
		logman.Error("ServiceForceUpdate failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "强制重部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Service force updated", nil)
}

// SwarmServiceLogs 获取服务日志
func (h *SwarmHandler) SwarmServiceLogs(c *gin.Context) {
	serviceID := c.Query("id")
	tail := c.DefaultQuery("tail", "100")
	if serviceID == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少服务 ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	reader, err := h.dockerClient.ServiceLogs(ctx, serviceID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("ServiceLogs failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取日志失败: "+err.Error())
		return
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "读取日志失败")
		return
	}

	// 去除 Docker multiplexed stream 头部（每行前 8 字节）
	lines := helper.ParseDockerLogs(raw)

	helper.RespondSuccess(c, "Logs retrieved", gin.H{"logs": lines})
}
