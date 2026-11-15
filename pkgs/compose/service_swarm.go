package compose

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/swarm"
)

// ==================== Compose service -> Swarm service spec ====================

// ServiceToSwarmSpec 将 compose ServiceConfig 转为 Docker SDK Swarm ServiceSpec。
func ServiceToSwarmSpec(project *types.Project, svc types.ServiceConfig) (swarm.ServiceSpec, error) {
	if svc.Image == "" {
		return swarm.ServiceSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := cmp.Or(svc.Name, svc.ContainerName)
	spec := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:          svc.Image,
				Env:            envToSlice(svc.Environment),
				Command:        []string(svc.Entrypoint),
				Args:           []string(svc.Command),
				Dir:            svc.WorkingDir,
				User:           svc.User,
				Hostname:       svc.Hostname,
				Hosts:          swarmHosts(svc.ExtraHosts),
				TTY:            svc.Tty,
				OpenStdin:      svc.StdinOpen,
				ReadOnly:       svc.ReadOnly,
				StopSignal:     svc.StopSignal,
				Sysctls:        map[string]string(svc.Sysctls),
				CapabilityAdd:  svc.CapAdd,
				CapabilityDrop: svc.CapDrop,
			},
		},
		EndpointSpec: &swarm.EndpointSpec{},

		// 内嵌 swarm.Annotations
		Name:   name,
		Labels: buildServiceLabels(project, svc, nil),
	}

	if len(svc.DNS) > 0 || len(svc.DNSOpts) > 0 || len(svc.DNSSearch) > 0 {
		spec.TaskTemplate.ContainerSpec.DNSConfig = &swarm.DNSConfig{
			Nameservers: []string(svc.DNS),
			Options:     svc.DNSOpts,
			Search:      []string(svc.DNSSearch),
		}
	}

	applySwarmMode(svc, &spec)
	applySwarmPolicies(svc, &spec)
	applySwarmNetworks(project, svc, &spec)
	applySwarmPorts(svc, &spec)
	applySwarmVolumes(svc, &spec)
	applySwarmPlacement(svc, &spec)
	applySwarmResources(svc, &spec)

	return spec, nil
}

// ─── 辅助函数 ───

// ==================== Swarm task/service mapping ====================

func applySwarmMode(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	replicas := uint64(1)
	if svc.Deploy != nil {
		if svc.Deploy.Mode == "global" {
			spec.Mode = swarm.ServiceMode{Global: &swarm.GlobalService{}}
			return
		}
		if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
			replicas = uint64(*svc.Deploy.Replicas)
		}
	}
	spec.Mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
}

func applySwarmNetworks(project *types.Project, svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	for _, k := range svc.NetworksByPriority() {
		network := swarm.NetworkAttachmentConfig{Target: resolveNetworkName(project, k)}
		if config := svc.Networks[k]; config != nil {
			network.Aliases = config.Aliases
		}
		spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, network)
	}
}

func applySwarmPolicies(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	if svc.Deploy == nil {
		return
	}
	if policy := svc.Deploy.RestartPolicy; policy != nil {
		spec.TaskTemplate.RestartPolicy = &swarm.RestartPolicy{
			Condition:   swarm.RestartPolicyCondition(policy.Condition),
			Delay:       (*time.Duration)(policy.Delay),
			MaxAttempts: policy.MaxAttempts,
			Window:      (*time.Duration)(policy.Window),
		}
	}
	spec.UpdateConfig = swarmUpdateConfig(svc.Deploy.UpdateConfig)
	spec.RollbackConfig = swarmUpdateConfig(svc.Deploy.RollbackConfig)
}

func applySwarmPorts(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		publishedStr := p.Published
		if _, after, ok := strings.CutLast(publishedStr, ":"); ok {
			publishedStr = after
		}
		published, _ := strconv.Atoi(publishedStr)
		if published == 0 {
			published = int(p.Target)
		}
		proto := swarm.PortConfigProtocolTCP
		if strings.EqualFold(p.Protocol, "udp") {
			proto = swarm.PortConfigProtocolUDP
		}
		publishMode := swarm.PortConfigPublishModeIngress
		if strings.EqualFold(p.Mode, "host") {
			publishMode = swarm.PortConfigPublishModeHost
		}
		spec.EndpointSpec.Ports = append(spec.EndpointSpec.Ports, swarm.PortConfig{
			PublishedPort: uint32(published),
			TargetPort:    p.Target,
			Protocol:      proto,
			PublishMode:   publishMode,
		})
	}
}

func applySwarmVolumes(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, serviceMounts(svc.Volumes)...)
}

func applySwarmPlacement(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	if svc.Deploy == nil || len(svc.Deploy.Placement.Constraints) == 0 {
		return
	}
	spec.TaskTemplate.Placement = &swarm.Placement{Constraints: svc.Deploy.Placement.Constraints}
}

func applySwarmResources(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	if svc.Deploy == nil {
		return
	}
	if limits := svc.Deploy.Resources.Limits; limits != nil {
		spec.TaskTemplate.Resources.Limits = &swarm.Limit{
			MemoryBytes: int64(limits.MemoryBytes),
			Pids:        limits.Pids,
		}
		if limits.NanoCPUs > 0 {
			spec.TaskTemplate.Resources.Limits.NanoCPUs = int64(float64(limits.NanoCPUs) * 1e9)
		}
	}
	if reservations := svc.Deploy.Resources.Reservations; reservations != nil {
		spec.TaskTemplate.Resources.Reservations = &swarm.Resources{
			MemoryBytes: int64(reservations.MemoryBytes),
		}
		if reservations.NanoCPUs > 0 {
			spec.TaskTemplate.Resources.Reservations.NanoCPUs = int64(float64(reservations.NanoCPUs) * 1e9)
		}
	}
}

// ==================== Swarm value converters ====================

func swarmUpdateConfig(config *types.UpdateConfig) *swarm.UpdateConfig {
	if config == nil {
		return nil
	}
	// Docker 的非指针字段用 0 表示全部并行；仅未填写时使用默认值 1。
	parallelism := uint64(1)
	if config.Parallelism != nil {
		parallelism = *config.Parallelism
	}
	return &swarm.UpdateConfig{
		Parallelism:     parallelism,
		Delay:           time.Duration(config.Delay),
		FailureAction:   config.FailureAction,
		Monitor:         time.Duration(config.Monitor),
		MaxFailureRatio: config.MaxFailureRatio,
		Order:           config.Order,
	}
}

func swarmHosts(hosts types.HostsList) []string {
	if len(hosts) == 0 {
		return nil
	}
	result := make([]string, 0, len(hosts))
	for host, ips := range hosts {
		for _, ip := range ips {
			if host != "" && ip != "" {
				result = append(result, ip+" "+host)
			}
		}
	}
	return result
}
