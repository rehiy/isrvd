// Package swarm 提供 Swarm 业务服务层
package swarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/mount"
	dockerswarm "github.com/docker/docker/api/types/swarm"

	"isrvd/internal/registry"
	pkgswarm "isrvd/pkgs/swarm"
)

// NodeInfo Swarm 节点信息（列表项），保持前端稳定响应结构。
type NodeInfo struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Role          string `json:"role"`
	Availability  string `json:"availability"`
	State         string `json:"state"`
	Addr          string `json:"addr"`
	EngineVersion string `json:"engineVersion"`
	Leader        bool   `json:"leader"`
}

// NodeDetail 节点详情，保持前端稳定响应结构。
type NodeDetail struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	Role          string            `json:"role"`
	Availability  string            `json:"availability"`
	State         string            `json:"state"`
	Addr          string            `json:"addr"`
	EngineVersion string            `json:"engineVersion"`
	Leader        bool              `json:"leader"`
	OS            string            `json:"os"`
	Architecture  string            `json:"architecture"`
	CPUs          int64             `json:"cpus"`
	MemoryBytes   int64             `json:"memoryBytes"`
	Labels        map[string]string `json:"labels"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

// Task Swarm 任务信息，保持前端稳定响应结构。
type Task struct {
	ID          string `json:"id"`
	ServiceID   string `json:"serviceID"`
	ServiceName string `json:"serviceName"`
	NodeID      string `json:"nodeID"`
	NodeName    string `json:"nodeName"`
	Slot        int    `json:"slot"`
	Image       string `json:"image"`
	State       string `json:"state"`
	Message     string `json:"message"`
	Err         string `json:"err"`
	UpdatedAt   string `json:"updatedAt"`
}

// ServiceInfo 服务列表信息（精简视图），保持前端稳定响应结构。
type ServiceInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Mode         string        `json:"mode"`
	Replicas     *uint64       `json:"replicas"`
	RunningTasks int           `json:"runningTasks"`
	Ports        []ServicePort `json:"ports"`
	CreatedAt    string        `json:"createdAt"`
	UpdatedAt    string        `json:"updatedAt"`
}

// ServiceDetail 服务详情（完整视图）。
type ServiceDetail struct {
	ServiceSpec
	ID           string `json:"id"`
	RunningTasks int    `json:"runningTasks"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ServiceSpec 服务可写配置（创建/更新共用），保持 HTTP API 兼容。
type ServiceSpec struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Mode        string            `json:"mode"`
	Replicas    *uint64           `json:"replicas"`
	Env         []string          `json:"env"`
	Args        []string          `json:"args"`
	Networks    []string          `json:"networks"`
	Ports       []ServicePort     `json:"ports"`
	Mounts      []ServiceMount    `json:"mounts"`
	Labels      map[string]string `json:"labels"`
	Constraints []string          `json:"constraints"`
}

// ServicePort 服务端口信息。
type ServicePort struct {
	Protocol      string `json:"protocol"`
	TargetPort    uint32 `json:"targetPort"`
	PublishedPort uint32 `json:"publishedPort"`
	PublishMode   string `json:"publishMode"`
}

// ServiceMount 服务挂载信息。
type ServiceMount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// Service Swarm 业务服务
type Service struct {
	svc *pkgswarm.SwarmService
}

// NewService 创建 Swarm 业务服务，验证节点是否是 Swarm manager
func NewService() (*Service, error) {
	svc := registry.SwarmService
	if svc == nil {
		return nil, fmt.Errorf("Swarm 服务未初始化")
	}
	// 验证节点是否加入 Swarm 且为 manager
	if _, err := svc.GetClient().SwarmInspect(context.Background()); err != nil {
		return nil, fmt.Errorf("Swarm 不可用: %w", err)
	}
	return &Service{svc: svc}, nil
}

// CheckAvailability 检测 Swarm 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.svc == nil {
		return false
	}
	_, err := s.svc.GetClient().SwarmInspect(ctx)
	return err == nil
}

// Info 获取 Swarm 集群概览
func (s *Service) Info(ctx context.Context) (map[string]any, error) {
	info, err := s.svc.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Swarm 信息失败: %w", err)
	}
	return info, nil
}

// JoinToken 获取加入集群的 token
func (s *Service) JoinToken(ctx context.Context) (map[string]string, error) {
	tokens, err := s.svc.JoinToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取加入令牌失败: %w", err)
	}
	return tokens, nil
}

// NodeList 获取节点列表
func (s *Service) NodeList(ctx context.Context) ([]NodeInfo, error) {
	list, err := s.svc.NodeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}
	result := make([]NodeInfo, 0, len(list))
	for _, node := range list {
		result = append(result, nodeInfoFromRaw(node))
	}
	return result, nil
}

// NodeAction 节点操作
func (s *Service) NodeAction(ctx context.Context, id, action string) error {
	if id == "" {
		return fmt.Errorf("节点 ID 不能为空")
	}
	if action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.svc.NodeAction(ctx, id, action); err != nil {
		return fmt.Errorf("节点操作 %s 失败: %w", action, err)
	}
	return nil
}

// NodeInspect 获取节点详情
func (s *Service) NodeInspect(ctx context.Context, id string) (*NodeDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少节点 ID")
	}
	node, err := s.svc.NodeInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取节点详情失败: %w", err)
	}
	return nodeDetailFromRaw(node), nil
}

// ServiceList 获取服务列表
func (s *Service) ServiceList(ctx context.Context) ([]ServiceInfo, error) {
	list, err := s.svc.ServiceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}
	runningMap := s.svc.ServiceRunningTasksMap(ctx)
	result := make([]ServiceInfo, 0, len(list))
	for _, svc := range list {
		result = append(result, serviceInfoFromRaw(svc, runningMap[svc.ID]))
	}
	return result, nil
}

// ServiceInspect 获取服务详情
func (s *Service) ServiceInspect(ctx context.Context, id string) (*ServiceDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	detail, err := s.svc.ServiceInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取服务详情失败: %w", err)
	}
	return serviceDetailFromRaw(detail, s.svc.ServiceRunningTasks(ctx, detail.ID)), nil
}

// ServiceCreate 创建服务。
func (s *Service) ServiceCreate(ctx context.Context, req ServiceSpec) (string, error) {
	if req.Name == "" {
		return "", fmt.Errorf("服务名称不能为空")
	}
	if req.Image == "" {
		return "", fmt.Errorf("镜像名称不能为空")
	}
	id, err := s.svc.ServiceCreate(ctx, serviceSpecToRaw(req))
	if err != nil {
		return "", fmt.Errorf("创建服务失败: %w", err)
	}
	return id, nil
}

// ServiceCreateRaw 使用 Docker SDK 原始结构创建服务，供 Compose 部署链路复用。
func (s *Service) ServiceCreateRaw(ctx context.Context, spec dockerswarm.ServiceSpec) (string, error) {
	if spec.Name == "" {
		return "", fmt.Errorf("服务名称不能为空")
	}
	if spec.TaskTemplate.ContainerSpec == nil || spec.TaskTemplate.ContainerSpec.Image == "" {
		return "", fmt.Errorf("镜像名称不能为空")
	}
	id, err := s.svc.ServiceCreate(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("创建服务失败: %w", err)
	}
	return id, nil
}

func serviceInfoFromRaw(svc dockerswarm.Service, runningTasks int) ServiceInfo {
	info := ServiceInfo{
		ID:           svc.ID,
		Name:         svc.Spec.Name,
		Mode:         "replicated",
		RunningTasks: runningTasks,
		CreatedAt:    svc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    svc.UpdatedAt.Format(time.RFC3339),
	}
	if svc.Spec.TaskTemplate.ContainerSpec != nil {
		info.Image = svc.Spec.TaskTemplate.ContainerSpec.Image
	}
	if svc.Spec.Mode.Global != nil {
		info.Mode = "global"
	} else if svc.Spec.Mode.Replicated != nil {
		info.Replicas = svc.Spec.Mode.Replicated.Replicas
	}
	for _, p := range svc.Endpoint.Ports {
		if p.PublishedPort == 0 && p.TargetPort == 0 {
			continue
		}
		info.Ports = append(info.Ports, ServicePort{
			Protocol:      string(p.Protocol),
			TargetPort:    p.TargetPort,
			PublishedPort: p.PublishedPort,
			PublishMode:   string(p.PublishMode),
		})
	}
	return info
}

func serviceDetailFromRaw(svc dockerswarm.Service, runningTasks int) *ServiceDetail {
	info := serviceInfoFromRaw(svc, runningTasks)
	detail := &ServiceDetail{
		ServiceSpec: ServiceSpec{
			Name:     info.Name,
			Image:    info.Image,
			Mode:     info.Mode,
			Replicas: info.Replicas,
		},
		ID:           info.ID,
		RunningTasks: info.RunningTasks,
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
	if svc.Spec.TaskTemplate.ContainerSpec != nil {
		detail.Env = svc.Spec.TaskTemplate.ContainerSpec.Env
		detail.Args = svc.Spec.TaskTemplate.ContainerSpec.Args
		detail.Labels = svc.Spec.Labels
		for _, mt := range svc.Spec.TaskTemplate.ContainerSpec.Mounts {
			detail.Mounts = append(detail.Mounts, ServiceMount{
				Type:     string(mt.Type),
				Source:   mt.Source,
				Target:   mt.Target,
				ReadOnly: mt.ReadOnly,
			})
		}
	}
	for _, n := range svc.Spec.TaskTemplate.Networks {
		detail.Networks = append(detail.Networks, n.Target)
	}
	if svc.Spec.TaskTemplate.Placement != nil {
		detail.Constraints = svc.Spec.TaskTemplate.Placement.Constraints
	}
	detail.Ports = info.Ports
	return detail
}

func serviceSpecToRaw(req ServiceSpec) dockerswarm.ServiceSpec {
	spec := dockerswarm.ServiceSpec{
		Annotations: dockerswarm.Annotations{Name: req.Name, Labels: req.Labels},
		TaskTemplate: dockerswarm.TaskSpec{
			ContainerSpec: &dockerswarm.ContainerSpec{
				Image: req.Image,
				Env:   req.Env,
				Args:  req.Args,
			},
		},
		EndpointSpec: &dockerswarm.EndpointSpec{},
	}
	if req.Mode == "global" {
		spec.Mode = dockerswarm.ServiceMode{Global: &dockerswarm.GlobalService{}}
	} else {
		replicas := uint64(1)
		if req.Replicas != nil && *req.Replicas > 0 {
			replicas = *req.Replicas
		}
		spec.Mode = dockerswarm.ServiceMode{Replicated: &dockerswarm.ReplicatedService{Replicas: &replicas}}
	}
	for _, p := range req.Ports {
		proto := dockerswarm.PortConfigProtocolTCP
		if strings.EqualFold(p.Protocol, "udp") {
			proto = dockerswarm.PortConfigProtocolUDP
		}
		publishMode := dockerswarm.PortConfigPublishModeIngress
		if strings.EqualFold(p.PublishMode, "host") {
			publishMode = dockerswarm.PortConfigPublishModeHost
		}
		spec.EndpointSpec.Ports = append(spec.EndpointSpec.Ports, dockerswarm.PortConfig{
			Protocol:      proto,
			PublishedPort: p.PublishedPort,
			TargetPort:    p.TargetPort,
			PublishMode:   publishMode,
		})
	}
	for _, mt := range req.Mounts {
		mountType := mount.TypeBind
		if mt.Type == "volume" {
			mountType = mount.TypeVolume
		}
		spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, mount.Mount{
			Type:     mountType,
			Source:   mt.Source,
			Target:   mt.Target,
			ReadOnly: mt.ReadOnly,
		})
	}
	for _, n := range req.Networks {
		spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, dockerswarm.NetworkAttachmentConfig{Target: n})
	}
	if len(req.Constraints) > 0 {
		spec.TaskTemplate.Placement = &dockerswarm.Placement{Constraints: req.Constraints}
	}
	return spec
}

// ServiceAction 服务操作
func (s *Service) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	if id == "" {
		return fmt.Errorf("服务 ID 不能为空")
	}
	if action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.svc.ServiceAction(ctx, id, action, replicas); err != nil {
		return fmt.Errorf("服务操作 %s 失败: %w", action, err)
	}
	return nil
}

// ServiceForceUpdate 强制重新部署服务
func (s *Service) ServiceForceUpdate(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("服务 ID 不能为空")
	}
	if err := s.svc.ServiceForceUpdate(ctx, id); err != nil {
		return fmt.Errorf("强制重新部署服务失败: %w", err)
	}
	return nil
}

// ServiceLogs 获取服务日志
func (s *Service) ServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("缺少服务 ID")
	}
	logs, err := s.svc.ServiceLogs(ctx, serviceID, tail)
	if err != nil {
		return nil, fmt.Errorf("获取服务日志失败: %w", err)
	}
	return logs, nil
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

func nodeInfoFromRaw(node dockerswarm.Node) NodeInfo {
	return NodeInfo{
		ID:            node.ID,
		Hostname:      node.Description.Hostname,
		Role:          string(node.Spec.Role),
		Availability:  string(node.Spec.Availability),
		State:         string(node.Status.State),
		Addr:          node.Status.Addr,
		EngineVersion: node.Description.Engine.EngineVersion,
		Leader:        node.ManagerStatus != nil && node.ManagerStatus.Leader,
	}
}

func nodeDetailFromRaw(node dockerswarm.Node) *NodeDetail {
	info := nodeInfoFromRaw(node)
	return &NodeDetail{
		ID: info.ID, Hostname: info.Hostname, Role: info.Role, Availability: info.Availability, State: info.State,
		Addr: info.Addr, EngineVersion: info.EngineVersion, Leader: info.Leader,
		OS: node.Description.Platform.OS, Architecture: node.Description.Platform.Architecture,
		CPUs: node.Description.Resources.NanoCPUs / 1e9, MemoryBytes: node.Description.Resources.MemoryBytes,
		Labels: node.Spec.Labels, CreatedAt: node.Meta.CreatedAt.Format(time.RFC3339), UpdatedAt: node.Meta.UpdatedAt.Format(time.RFC3339),
	}
}

func tasksFromRaw(tasks []dockerswarm.Task, services []dockerswarm.Service, nodes []dockerswarm.Node) []Task {
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
