package swarm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/docker"
)

// SwarmService Swarm 服务信息
type SwarmService struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Image        string      `json:"image"`
	Mode         string      `json:"mode"`
	Replicas     *uint64     `json:"replicas"`
	RunningTasks int         `json:"runningTasks"`
	Ports        interface{} `json:"ports"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

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

		var ports []SwarmServicePort
		for _, p := range s.Endpoint.Ports {
			if p.PublishedPort == 0 && p.TargetPort == 0 {
				continue
			}
			ports = append(ports, SwarmServicePort{
				Protocol:      string(p.Protocol),
				TargetPort:    p.TargetPort,
				PublishedPort: p.PublishedPort,
				PublishMode:   string(p.PublishMode),
			})
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

// SwarmPortConfig Swarm 端口配置
type SwarmPortConfig struct {
	Published int    `json:"published"`
	Target    int    `json:"target"`
	Protocol  string `json:"protocol"`
}

// SwarmMount Swarm 挂载配置
type SwarmMount struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// SwarmCreateServiceRequest Swarm 创建服务请求
type SwarmCreateServiceRequest struct {
	Name     string            `json:"name" binding:"required"`
	Image    string            `json:"image" binding:"required"`
	Mode     string            `json:"mode"`
	Replicas int               `json:"replicas"`
	Env      []string          `json:"env"`
	Args     []string          `json:"args"`
	Networks []string          `json:"networks"`
	Ports    []SwarmPortConfig `json:"ports"`
	Mounts   []SwarmMount      `json:"mounts"`
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
	reader, err := m.client.ServiceLogs(ctx, serviceID, container.LogsOptions{
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

// SwarmServicePort 服务端口信息
type SwarmServicePort struct {
	Protocol      string `json:"protocol"`
	TargetPort    uint32 `json:"targetPort"`
	PublishedPort uint32 `json:"publishedPort"`
	PublishMode   string `json:"publishMode"`
}

// SwarmServiceMount 服务挂载信息
type SwarmServiceMount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// SwarmServiceInspect 服务详情
type SwarmServiceInspect struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Mode         string              `json:"mode"`
	Replicas     *uint64             `json:"replicas"`
	RunningTasks int                 `json:"runningTasks"`
	Ports        []SwarmServicePort  `json:"ports"`
	Env          []string            `json:"env"`
	Args         []string            `json:"args"`
	Networks     []string            `json:"networks"`
	Mounts       []SwarmServiceMount `json:"mounts"`
	Labels       map[string]string   `json:"labels"`
	Constraints  []string            `json:"constraints"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
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
