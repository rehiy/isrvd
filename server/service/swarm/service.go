// Package swarm 提供 Swarm 业务服务层
package swarm

import (
	"context"
	"fmt"
	"io"
	pkgSwarm "isrvd/pkgs/swarm"
	"strings"
	"time"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"

	"isrvd/pkgs/docker"
)

// Service Swarm 业务服务
type Service struct {
	svc *pkgSwarm.SwarmService
}

// NewService 创建 Swarm 业务服务，验证节点是否是 Swarm manager。
// dockerRaw 由调用方注入，为 nil 表示 Docker 不可用。
func NewService(ctx context.Context, dockerRaw *docker.DockerService) (*Service, error) {
	if dockerRaw == nil {
		return nil, fmt.Errorf("Docker 服务未初始化")
	}
	svc := pkgSwarm.NewSwarmService(dockerRaw.Client(), dockerRaw.RegistryAuth)
	// 验证节点是否加入 Swarm 且为 manager
	if _, err := svc.Client().SwarmInspect(ctx); err != nil {
		return nil, fmt.Errorf("Swarm 不可用: %w", err)
	}
	return &Service{svc: svc}, nil
}

// CheckAvailability 检测 Swarm 可用性
func (s *Service) CheckAvailability(ctx context.Context) bool {
	if s.svc == nil {
		return false
	}
	_, err := s.svc.Client().SwarmInspect(ctx)
	return err == nil
}

// Raw 返回底层 SwarmService，供 compose 服务复用。
// s 为 nil（Swarm 不可用）时安全返回 nil。
func (s *Service) Raw() *pkgSwarm.SwarmService {
	if s == nil {
		return nil
	}
	return s.svc
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

// ServicePort 服务端口信息。
type ServicePort struct {
	Protocol      string `json:"protocol"`      // 协议（tcp/udp）
	TargetPort    uint32 `json:"targetPort"`    // 容器内端口
	PublishedPort uint32 `json:"publishedPort"` // 对外发布端口
	PublishMode   string `json:"publishMode"`   // 发布模式（ingress/host）
}

// ServiceInfo 服务列表信息（精简视图），保持前端稳定响应结构。
type ServiceInfo struct {
	ID           string        `json:"id"`           // 服务 ID
	Name         string        `json:"name"`         // 服务名称
	Image        string        `json:"image"`        // 镜像名称
	Mode         string        `json:"mode"`         // 部署模式（replicated/global）
	Replicas     *uint64       `json:"replicas"`     // 副本数
	RunningTasks int           `json:"runningTasks"` // 运行中的任务数
	Ports        []ServicePort `json:"ports"`        // 端口映射列表
	CreatedAt    string        `json:"createdAt"`    // 创建时间
	UpdatedAt    string        `json:"updatedAt"`    // 更新时间
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

// ServiceMount 服务挂载信息。
type ServiceMount struct {
	Type     string `json:"type"`     // 挂载类型（volume/bind）
	Source   string `json:"source"`   // 来源（卷名或宿主机路径）
	Target   string `json:"target"`   // 容器内挂载路径
	ReadOnly bool   `json:"readOnly"` // 是否只读
}

// ServiceSpec 服务可写配置（创建/更新共用），保持 HTTP API 兼容。
type ServiceSpec struct {
	Name        string            `json:"name"`        // 服务名称
	Image       string            `json:"image"`       // 镜像名称
	Mode        string            `json:"mode"`        // 部署模式（replicated/global）
	Replicas    *uint64           `json:"replicas"`    // 副本数
	Env         []string          `json:"env"`         // 环境变量列表
	Args        []string          `json:"args"`        // 启动参数
	Networks    []string          `json:"networks"`    // 网络列表
	Ports       []ServicePort     `json:"ports"`       // 端口映射
	Mounts      []ServiceMount    `json:"mounts"`      // 挂载卷
	Labels      map[string]string `json:"labels"`      // 标签
	Constraints []string          `json:"constraints"` // 调度约束
}

// ServiceDetail 服务详情（完整视图）。
type ServiceDetail struct {
	ServiceSpec
	ID           string `json:"id"`           // 服务 ID
	RunningTasks int    `json:"runningTasks"` // 运行中的任务数
	CreatedAt    string `json:"createdAt"`    // 创建时间
	UpdatedAt    string `json:"updatedAt"`    // 更新时间
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

// ServiceLogsStream 服务实时日志流
func (s *Service) ServiceLogsStream(ctx context.Context, w io.Writer, serviceID, tail string) {
	s.svc.ServiceLogsStream(ctx, w, serviceID, tail)
}

func serviceInfoFromRaw(svc swarm.Service, runningTasks int) ServiceInfo {
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

func serviceDetailFromRaw(svc swarm.Service, runningTasks int) *ServiceDetail {
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

func serviceSpecToRaw(req ServiceSpec) swarm.ServiceSpec {
	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{Name: req.Name, Labels: req.Labels},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: req.Image,
				Env:   req.Env,
				Args:  req.Args,
			},
		},
		EndpointSpec: &swarm.EndpointSpec{},
	}
	if req.Mode == "global" {
		spec.Mode = swarm.ServiceMode{Global: &swarm.GlobalService{}}
	} else {
		replicas := uint64(1)
		if req.Replicas != nil && *req.Replicas > 0 {
			replicas = *req.Replicas
		}
		spec.Mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
	}
	for _, p := range req.Ports {
		proto := swarm.PortConfigProtocolTCP
		if strings.EqualFold(p.Protocol, "udp") {
			proto = swarm.PortConfigProtocolUDP
		}
		publishMode := swarm.PortConfigPublishModeIngress
		if strings.EqualFold(p.PublishMode, "host") {
			publishMode = swarm.PortConfigPublishModeHost
		}
		spec.EndpointSpec.Ports = append(spec.EndpointSpec.Ports, swarm.PortConfig{
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
		spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, swarm.NetworkAttachmentConfig{Target: n})
	}
	if len(req.Constraints) > 0 {
		spec.TaskTemplate.Placement = &swarm.Placement{Constraints: req.Constraints}
	}
	return spec
}
