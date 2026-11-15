package compose

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

// ==================== Compose service -> Swarm service spec ====================

// ServiceToSwarmSpec 将 compose ServiceConfig 转为 Docker SDK Swarm ServiceSpec。
func ServiceToSwarmSpec(project *types.Project, svc types.ServiceConfig) (swarm.ServiceSpec, error) {
	if svc.Image == "" {
		return swarm.ServiceSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}
	if len(svc.Secrets) > 0 || len(svc.Configs) > 0 {
		return swarm.ServiceSpec{}, fmt.Errorf("service %q: Swarm configs 和 secrets 暂不支持", svc.Name)
	}
	if svc.Deploy != nil && svc.Deploy.Mode != "" && svc.Deploy.Mode != "global" && svc.Deploy.Mode != "replicated" {
		return swarm.ServiceSpec{}, fmt.Errorf("service %q: 不支持 Swarm 模式 %s", svc.Name, svc.Deploy.Mode)
	}

	name := cmp.Or(svc.Name, svc.ContainerName)
	spec := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:           svc.Image,
				Env:             envToSlice(svc.Environment),
				Command:         []string(svc.Entrypoint),
				Args:            []string(svc.Command),
				Dir:             svc.WorkingDir,
				User:            svc.User,
				Groups:          svc.GroupAdd,
				Init:            svc.Init,
				StopGracePeriod: (*time.Duration)(svc.StopGracePeriod),
				Labels:          svc.Labels,
				Ulimits:         dockerUlimits(svc.Ulimits),
				OomScoreAdj:     svc.OomScoreAdj,
				Isolation:       container.Isolation(svc.Isolation),
				Hostname:        svc.Hostname,
				Hosts:           swarmHosts(svc.ExtraHosts),
				TTY:             svc.Tty,
				OpenStdin:       svc.StdinOpen,
				ReadOnly:        svc.ReadOnly,
				StopSignal:      svc.StopSignal,
				Sysctls:         map[string]string(svc.Sysctls),
				CapabilityAdd:   svc.CapAdd,
				CapabilityDrop:  svc.CapDrop,
			},
		},
		EndpointSpec: &swarm.EndpointSpec{},

		// 内嵌 swarm.Annotations
		Name:   name,
		Labels: buildServiceLabels(project, svc, nil),
	}
	if svc.HealthCheck != nil {
		h := svc.HealthCheck
		check := &container.HealthConfig{Test: h.Test}
		if h.Disable {
			check.Test = []string{"NONE"}
		}
		if h.Interval != nil {
			check.Interval = time.Duration(*h.Interval)
		}
		if h.Timeout != nil {
			check.Timeout = time.Duration(*h.Timeout)
		}
		if h.StartPeriod != nil {
			check.StartPeriod = time.Duration(*h.StartPeriod)
		}
		if h.StartInterval != nil {
			check.StartInterval = time.Duration(*h.StartInterval)
		}
		if h.Retries != nil {
			if *h.Retries > uint64(^uint(0)>>1) {
				return swarm.ServiceSpec{}, fmt.Errorf("service %q: healthcheck retries 超出范围", svc.Name)
			}
			check.Retries = int(*h.Retries)
		}
		spec.TaskTemplate.ContainerSpec.Healthcheck = check
	}
	if svc.Logging != nil {
		spec.TaskTemplate.LogDriver = &swarm.Driver{Name: svc.Logging.Driver, Options: svc.Logging.Options}
	}
	if svc.Deploy != nil {
		for key, value := range svc.Deploy.Labels {
			spec.Labels[key] = value
		}
		if project != nil && project.Name != "" {
			spec.Labels[ComposeProjectLabel] = project.Name
		}
		spec.Labels[ComposeServiceLabel] = svc.Name
		spec.EndpointSpec.Mode = swarm.ResolutionMode(svc.Deploy.EndpointMode)
	}

	if len(svc.DNS) > 0 || len(svc.DNSOpts) > 0 || len(svc.DNSSearch) > 0 {
		spec.TaskTemplate.ContainerSpec.DNSConfig = &swarm.DNSConfig{
			Nameservers: []string(svc.DNS),
			Options:     svc.DNSOpts,
			Search:      []string(svc.DNSSearch),
		}
	}

	applySwarmMode(svc, &spec)
	if err := applySwarmPolicies(svc, &spec); err != nil {
		return swarm.ServiceSpec{}, err
	}
	applySwarmNetworks(project, svc, &spec)
	applySwarmPorts(svc, &spec)
	applySwarmVolumes(project, svc, &spec)
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
		if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas >= 0 {
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

func applySwarmPolicies(svc types.ServiceConfig, spec *swarm.ServiceSpec) error {
	if svc.Deploy == nil || svc.Deploy.RestartPolicy == nil {
		policy, attempts, hasAttempts := strings.Cut(svc.Restart, ":")
		switch policy {
		case "always", "unless-stopped":
			spec.TaskTemplate.RestartPolicy = &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionAny}
		case "on-failure":
			spec.TaskTemplate.RestartPolicy = &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionOnFailure}
			if hasAttempts {
				maxAttempts, err := strconv.ParseUint(attempts, 10, 64)
				if err != nil {
					return fmt.Errorf("service %q: 无效的重启次数: %w", svc.Name, err)
				}
				spec.TaskTemplate.RestartPolicy.MaxAttempts = &maxAttempts
			}
		case "no":
			spec.TaskTemplate.RestartPolicy = &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionNone}
		case "":
		default:
			return fmt.Errorf("service %q: 不支持重启策略 %s", svc.Name, svc.Restart)
		}
		if hasAttempts && policy != "on-failure" {
			return fmt.Errorf("service %q: 不支持重启策略 %s", svc.Name, svc.Restart)
		}
	}
	if svc.Deploy == nil {
		return nil
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
	return nil
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

func applySwarmVolumes(project *types.Project, svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	for _, volume := range svc.Volumes {
		if volume.Source == "" || volume.Target == "" {
			continue
		}
		m := mount.Mount{Type: mount.Type(volume.Type), Source: volume.Source, Target: volume.Target, ReadOnly: volume.ReadOnly, Consistency: mount.Consistency(volume.Consistency)}
		if volume.Type == types.VolumeTypeVolume && project != nil {
			if config, ok := project.Volumes[volume.Source]; ok && config.Name != "" {
				m.Source = config.Name
			}
		}
		if volume.Bind != nil {
			m.BindOptions = &mount.BindOptions{Propagation: mount.Propagation(volume.Bind.Propagation), CreateMountpoint: bool(volume.Bind.CreateHostPath)}
		}
		if volume.Volume != nil {
			m.VolumeOptions = &mount.VolumeOptions{NoCopy: volume.Volume.NoCopy, Subpath: volume.Volume.Subpath, Labels: map[string]string(volume.Volume.Labels)}
		}
		spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, m)
	}
}

func applySwarmPlacement(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	if svc.Deploy == nil {
		return
	}
	p := svc.Deploy.Placement
	if len(p.Constraints) == 0 && len(p.Preferences) == 0 && p.MaxReplicas == 0 {
		return
	}
	spec.TaskTemplate.Placement = &swarm.Placement{Constraints: p.Constraints, MaxReplicas: p.MaxReplicas}
	for _, preference := range p.Preferences {
		if preference.Spread != "" {
			spec.TaskTemplate.Placement.Preferences = append(spec.TaskTemplate.Placement.Preferences, swarm.PlacementPreference{Spread: &swarm.SpreadOver{SpreadDescriptor: preference.Spread}})
		}
	}
}

func applySwarmResources(svc types.ServiceConfig, spec *swarm.ServiceSpec) {
	if svc.Deploy == nil {
		return
	}
	if svc.Deploy.Resources.Limits != nil || svc.Deploy.Resources.Reservations != nil {
		spec.TaskTemplate.Resources = &swarm.ResourceRequirements{}
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
	slices.Sort(result)
	return result
}
