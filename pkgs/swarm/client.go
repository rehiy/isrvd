package swarm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/docker"
)

// SwarmManager Swarm 业务逻辑管理器
type SwarmManager struct {
	client *client.Client
}

// NewSwarmManager 创建 Swarm 管理器
func NewSwarmManager(dockerClient *client.Client) *SwarmManager {
	return &SwarmManager{client: dockerClient}
}

// GetClient 获取 Docker 客户端
func (m *SwarmManager) GetClient() *client.Client {
	return m.client
}

// --- 集群信息 ---

// GetSwarmInfo 获取 Swarm 集群概览
func (m *SwarmManager) GetSwarmInfo(ctx context.Context) (map[string]interface{}, error) {
	info, err := m.client.SwarmInspect(ctx)
	if err != nil {
		logman.Error("SwarmInspect failed", "error", err)
		return nil, err
	}

	nodes, _ := m.client.NodeList(ctx, types.NodeListOptions{})
	services, _ := m.client.ServiceList(ctx, types.ServiceListOptions{})
	tasks, _ := m.client.TaskList(ctx, types.TaskListOptions{})

	var managers, workers int
	for _, n := range nodes {
		if n.Spec.Role == swarm.NodeRoleManager {
			managers++
		} else {
			workers++
		}
	}

	return map[string]interface{}{
		"clusterID": info.ID,
		"createdAt": info.Meta.CreatedAt.Format(time.RFC3339),
		"nodes":     len(nodes),
		"managers":  managers,
		"workers":   workers,
		"services":  len(services),
		"tasks":     len(tasks),
	}, nil
}

// --- 节点管理 ---

// ListNodes 获取节点列表
func (m *SwarmManager) ListNodes(ctx context.Context) ([]SwarmNode, error) {
	nodes, err := m.client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logman.Error("NodeList failed", "error", err)
		return nil, err
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

	return result, nil
}

// NodeAction 节点操作（drain/active/pause/remove）
func (m *SwarmManager) NodeAction(ctx context.Context, id, action string) error {
	if action == "remove" {
		if err := m.client.NodeRemove(ctx, id, types.NodeRemoveOptions{Force: true}); err != nil {
			return err
		}
		return nil
	}

	node, _, err := m.client.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return err
	}

	switch action {
	case "drain":
		node.Spec.Availability = swarm.NodeAvailabilityDrain
	case "active":
		node.Spec.Availability = swarm.NodeAvailabilityActive
	case "pause":
		node.Spec.Availability = swarm.NodeAvailabilityPause
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	if err := m.client.NodeUpdate(ctx, id, node.Version, node.Spec); err != nil {
		logman.Error("NodeUpdate failed", "error", err)
		return err
	}

	return nil
}

// --- 服务管理 ---

// ListServices 获取服务列表
func (m *SwarmManager) ListServices(ctx context.Context) ([]SwarmService, error) {
	services, err := m.client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		logman.Error("ServiceList failed", "error", err)
		return nil, err
	}

	// 统计各服务运行中的任务数
	tasks, _ := m.client.TaskList(ctx, types.TaskListOptions{})
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

		var ports []interface{}
		for _, p := range s.Endpoint.Ports {
			if p.PublishedPort > 0 {
				ports = append(ports, map[string]interface{}{
					"published": p.PublishedPort,
					"target":    p.TargetPort,
					"protocol":  string(p.Protocol),
				})
			}
		}
		svc.Ports = ports

		result = append(result, svc)
	}

	return result, nil
}

// ServiceAction 服务操作（scale/remove）
func (m *SwarmManager) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	if action == "remove" {
		if err := m.client.ServiceRemove(ctx, id); err != nil {
			return err
		}
		return nil
	}

	if action == "scale" && replicas != nil {
		svc, _, err := m.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{InsertDefaults: true})
		if err != nil {
			return err
		}
		if svc.Spec.Mode.Replicated == nil {
			return fmt.Errorf("仅 replicated 模式服务支持 scale")
		}
		svc.Spec.Mode.Replicated.Replicas = replicas
		if _, err := m.client.ServiceUpdate(ctx, id, svc.Version, svc.Spec, types.ServiceUpdateOptions{}); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("不支持的操作: %s", action)
}

// CreateService 创建服务
func (m *SwarmManager) CreateService(ctx context.Context, req SwarmCreateServiceRequest) (string, error) {
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
	for _, mt := range req.Mounts {
		mountType := mount.TypeBind
		if mt.Type == "volume" {
			mountType = mount.TypeVolume
		}
		spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, mount.Mount{
			Type:   mountType,
			Source: mt.Source,
			Target: mt.Target,
		})
	}

	// 网络
	for _, n := range req.Networks {
		spec.Networks = append(spec.Networks, swarm.NetworkAttachmentConfig{Target: n})
	}

	resp, err := m.client.ServiceCreate(ctx, spec, types.ServiceCreateOptions{})
	if err != nil {
		logman.Error("ServiceCreate failed", "error", err)
		return "", err
	}

	return resp.ID, nil
}

// ForceUpdateService 强制重新部署服务
func (m *SwarmManager) ForceUpdateService(ctx context.Context, id string) error {
	svc, _, err := m.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		return err
	}

	svc.Spec.TaskTemplate.ForceUpdate++

	if _, err := m.client.ServiceUpdate(ctx, id, svc.Version, svc.Spec, types.ServiceUpdateOptions{}); err != nil {
		logman.Error("ServiceForceUpdate failed", "error", err)
		return err
	}

	return nil
}

// GetServiceLogs 获取服务日志
func (m *SwarmManager) GetServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	reader, err := m.client.ServiceLogs(ctx, serviceID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("ServiceLogs failed", "error", err)
		return nil, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return docker.ParseDockerLogs(raw), nil
}

// --- 任务管理 ---

// ListTasks 获取任务列表
func (m *SwarmManager) ListTasks(ctx context.Context, serviceID string) ([]SwarmTask, error) {
	opts := types.TaskListOptions{}
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
	services, _ := m.client.ServiceList(ctx, types.ServiceListOptions{})
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

	return result, nil
}

// InspectNode 获取节点详情
func (m *SwarmManager) InspectNode(ctx context.Context, id string) (*SwarmNodeInspect, error) {
	node, _, err := m.client.NodeInspectWithRaw(ctx, id)
	if err != nil {
		logman.Error("NodeInspect failed", "id", id, "error", err)
		return nil, err
	}

	result := &SwarmNodeInspect{
		ID:            node.ID,
		Hostname:      node.Description.Hostname,
		Role:          string(node.Spec.Role),
		Availability:  string(node.Spec.Availability),
		State:         string(node.Status.State),
		Addr:          node.Status.Addr,
		EngineVersion: node.Description.Engine.EngineVersion,
		Leader:        node.ManagerStatus != nil && node.ManagerStatus.Leader,
		OS:            node.Description.Platform.OS,
		Architecture:  node.Description.Platform.Architecture,
		CPUs:          node.Description.Resources.NanoCPUs / 1e9,
		MemoryBytes:   node.Description.Resources.MemoryBytes,
		Labels:        node.Spec.Labels,
		CreatedAt:     node.Meta.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     node.Meta.UpdatedAt.Format(time.RFC3339),
	}

	return result, nil
}

// InspectService 获取服务详情
func (m *SwarmManager) InspectService(ctx context.Context, id string) (*SwarmServiceInspect, error) {
	svc, _, err := m.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		logman.Error("ServiceInspect failed", "id", id, "error", err)
		return nil, err
	}

	// 统计运行中任务数
	f := filters.NewArgs()
	f.Add("service", svc.ID)
	tasks, _ := m.client.TaskList(ctx, types.TaskListOptions{Filters: f})
	runningTasks := 0
	for _, t := range tasks {
		if t.Status.State == swarm.TaskStateRunning {
			runningTasks++
		}
	}

	result := &SwarmServiceInspect{
		ID:           svc.ID,
		Name:         svc.Spec.Name,
		Image:        svc.Spec.TaskTemplate.ContainerSpec.Image,
		Mode:         "replicated",
		RunningTasks: runningTasks,
		Env:          svc.Spec.TaskTemplate.ContainerSpec.Env,
		Args:         svc.Spec.TaskTemplate.ContainerSpec.Args,
		Labels:       svc.Spec.Labels,
		CreatedAt:    svc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    svc.UpdatedAt.Format(time.RFC3339),
	}

	if svc.Spec.Mode.Global != nil {
		result.Mode = "global"
	} else if svc.Spec.Mode.Replicated != nil {
		result.Replicas = svc.Spec.Mode.Replicated.Replicas
	}

	// 端口
	for _, p := range svc.Endpoint.Ports {
		result.Ports = append(result.Ports, SwarmServicePort{
			Protocol:      string(p.Protocol),
			TargetPort:    p.TargetPort,
			PublishedPort: p.PublishedPort,
			PublishMode:   string(p.PublishMode),
		})
	}

	// 挂载
	for _, mt := range svc.Spec.TaskTemplate.ContainerSpec.Mounts {
		result.Mounts = append(result.Mounts, SwarmServiceMount{
			Type:     string(mt.Type),
			Source:   mt.Source,
			Target:   mt.Target,
			ReadOnly: mt.ReadOnly,
		})
	}

	// 网络
	for _, n := range svc.Spec.Networks {
		result.Networks = append(result.Networks, n.Target)
	}

	// 约束
	if svc.Spec.TaskTemplate.Placement != nil {
		result.Constraints = svc.Spec.TaskTemplate.Placement.Constraints
	}

	return result, nil
}
