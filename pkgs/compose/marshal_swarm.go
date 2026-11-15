package compose

import (
	"cmp"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

// ==================== Swarm inspect -> Compose ====================

// ProjectFromSwarmInspect 将 swarm Service 原始配置反推为单服务 compose Project。
// containerDir 非空时，位于该目录内的 bind source 会输出为相对路径。
func ProjectFromSwarmInspect(svc swarm.Service, containerDir string) (*types.Project, error) {
	spec := svc.Spec
	cs := spec.TaskTemplate.ContainerSpec
	if cs == nil || cs.Image == "" {
		return nil, fmt.Errorf("swarm 服务数据不完整")
	}
	if len(cs.Secrets) > 0 || len(cs.Configs) > 0 {
		return nil, fmt.Errorf("无法从运行态安全还原 Swarm configs 或 secrets，请使用原始 Compose 文件")
	}

	name := cmp.Or(spec.Name, svc.ID)
	composeSvc := types.ServiceConfig{
		Name: name,

		// 内嵌 types.ContainerSpec
		Image:           cs.Image,
		Environment:     sliceToEnv(cs.Env),
		Entrypoint:      types.ShellCommand(cs.Command),
		Command:         types.ShellCommand(cs.Args),
		WorkingDir:      cs.Dir,
		User:            cs.User,
		Hostname:        cs.Hostname,
		ExtraHosts:      swarmExtraHostsToMap(cs.Hosts),
		ReadOnly:        cs.ReadOnly,
		StopSignal:      cs.StopSignal,
		Sysctls:         types.Mapping(cs.Sysctls),
		CapAdd:          cs.CapabilityAdd,
		CapDrop:         cs.CapabilityDrop,
		Labels:          cs.Labels,
		Ulimits:         inspectUlimits(cs.Ulimits),
		GroupAdd:        cs.Groups,
		Init:            cs.Init,
		StopGracePeriod: (*types.Duration)(cs.StopGracePeriod),
		OomScoreAdj:     cs.OomScoreAdj,
		Isolation:       string(cs.Isolation),

		// 内嵌 types.WorkloadSpec
		Tty:       cs.TTY,
		StdinOpen: cs.OpenStdin,
	}

	if cs.DNSConfig != nil {
		composeSvc.DNS = types.StringList(cs.DNSConfig.Nameservers)
		composeSvc.DNSOpts = cs.DNSConfig.Options
		composeSvc.DNSSearch = types.StringList(cs.DNSConfig.Search)
	}

	if spec.Mode.Global != nil {
		composeSvc.Deploy = &types.DeployConfig{Mode: "global"}
	} else if spec.Mode.Replicated != nil {
		r := 1
		if spec.Mode.Replicated.Replicas != nil {
			r = int(*spec.Mode.Replicated.Replicas)
		}
		composeSvc.Deploy = &types.DeployConfig{Mode: "replicated", Replicas: &r}
	}
	if cs.Healthcheck != nil {
		h := cs.Healthcheck
		check := &types.HealthCheckConfig{Test: types.HealthCheckTest(h.Test)}
		if h.Interval != 0 {
			value := types.Duration(h.Interval)
			check.Interval = &value
		}
		if h.Timeout != 0 {
			value := types.Duration(h.Timeout)
			check.Timeout = &value
		}
		if h.StartPeriod != 0 {
			value := types.Duration(h.StartPeriod)
			check.StartPeriod = &value
		}
		if h.StartInterval != 0 {
			value := types.Duration(h.StartInterval)
			check.StartInterval = &value
		}
		if h.Retries > 0 {
			value := uint64(h.Retries)
			check.Retries = &value
		}
		composeSvc.HealthCheck = check
	}
	if spec.TaskTemplate.LogDriver != nil {
		composeSvc.Logging = &types.LoggingConfig{Driver: spec.TaskTemplate.LogDriver.Name, Options: types.Options(spec.TaskTemplate.LogDriver.Options)}
	}
	if composeSvc.Deploy == nil && (len(spec.Labels) > 0 || spec.TaskTemplate.Resources != nil || spec.TaskTemplate.Placement != nil || (spec.EndpointSpec != nil && spec.EndpointSpec.Mode != "")) {
		composeSvc.Deploy = &types.DeployConfig{}
	}
	if composeSvc.Deploy != nil {
		composeSvc.Deploy.Labels = maps.Clone(spec.Labels)
		delete(composeSvc.Deploy.Labels, SwarmUpdateTokenLabel)
		if spec.EndpointSpec != nil {
			composeSvc.Deploy.EndpointMode = string(spec.EndpointSpec.Mode)
		}
		if resources := spec.TaskTemplate.Resources; resources != nil {
			if limits := resources.Limits; limits != nil {
				composeSvc.Deploy.Resources.Limits = &types.Resource{NanoCPUs: types.NanoCPUs(float64(limits.NanoCPUs) / 1e9), MemoryBytes: types.UnitBytes(limits.MemoryBytes), Pids: limits.Pids}
			}
			if reservations := resources.Reservations; reservations != nil {
				composeSvc.Deploy.Resources.Reservations = &types.Resource{NanoCPUs: types.NanoCPUs(float64(reservations.NanoCPUs) / 1e9), MemoryBytes: types.UnitBytes(reservations.MemoryBytes)}
			}
		}
		if placement := spec.TaskTemplate.Placement; placement != nil {
			composeSvc.Deploy.Placement.MaxReplicas = placement.MaxReplicas
			for _, preference := range placement.Preferences {
				if preference.Spread != nil {
					composeSvc.Deploy.Placement.Preferences = append(composeSvc.Deploy.Placement.Preferences, types.PlacementPreferences{Spread: preference.Spread.SpreadDescriptor})
				}
			}
		}
	}
	if spec.TaskTemplate.Placement != nil && len(spec.TaskTemplate.Placement.Constraints) > 0 {
		if composeSvc.Deploy == nil {
			composeSvc.Deploy = &types.DeployConfig{}
		}
		composeSvc.Deploy.Placement.Constraints = spec.TaskTemplate.Placement.Constraints
	}
	if spec.TaskTemplate.RestartPolicy != nil || spec.UpdateConfig != nil || spec.RollbackConfig != nil {
		if composeSvc.Deploy == nil {
			composeSvc.Deploy = &types.DeployConfig{}
		}
		if policy := spec.TaskTemplate.RestartPolicy; policy != nil {
			composeSvc.Deploy.RestartPolicy = &types.RestartPolicy{
				Condition:   string(policy.Condition),
				Delay:       (*types.Duration)(policy.Delay),
				MaxAttempts: policy.MaxAttempts,
				Window:      (*types.Duration)(policy.Window),
			}
		}
		composeSvc.Deploy.UpdateConfig = swarmUpdateConfigToCompose(spec.UpdateConfig)
		composeSvc.Deploy.RollbackConfig = swarmUpdateConfigToCompose(spec.RollbackConfig)
	}

	if spec.EndpointSpec != nil {
		for _, p := range spec.EndpointSpec.Ports {
			composeSvc.Ports = append(composeSvc.Ports, types.ServicePortConfig{
				Target:    p.TargetPort,
				Published: strconv.Itoa(int(p.PublishedPort)),
				Protocol:  cmp.Or(string(p.Protocol), "tcp"),
				Mode:      cmp.Or(string(p.PublishMode), "ingress"),
			})
		}
	}

	projectVolumes := types.Volumes{}
	for _, m := range cs.Mounts {
		volume := types.ServiceVolumeConfig{
			Type:        string(m.Type),
			Source:      swarmMountSource(m, containerDir),
			Target:      m.Target,
			ReadOnly:    m.ReadOnly,
			Consistency: string(m.Consistency),
		}
		if m.BindOptions != nil {
			volume.Bind = &types.ServiceVolumeBind{Propagation: string(m.BindOptions.Propagation), CreateHostPath: types.OptOut(m.BindOptions.CreateMountpoint)}
		}
		if m.VolumeOptions != nil {
			volume.Volume = &types.ServiceVolumeVolume{NoCopy: m.VolumeOptions.NoCopy, Subpath: m.VolumeOptions.Subpath, Labels: types.Mapping(m.VolumeOptions.Labels)}
		}
		if m.Type == mount.TypeVolume && m.Source != "" {
			projectVolumes[m.Source] = types.VolumeConfig{Name: m.Source, External: true}
		}
		composeSvc.Volumes = append(composeSvc.Volumes, volume)
	}

	var projectNetworks types.Networks
	if len(spec.TaskTemplate.Networks) > 0 {
		composeSvc.Networks = make(map[string]*types.ServiceNetworkConfig, len(spec.TaskTemplate.Networks))
		projectNetworks = make(types.Networks, len(spec.TaskTemplate.Networks))
		for _, n := range spec.TaskTemplate.Networks {
			netCfg := &types.ServiceNetworkConfig{Aliases: n.Aliases}
			composeSvc.Networks[n.Target] = netCfg
			projectNetworks[n.Target] = types.NetworkConfig{Name: n.Target, Driver: "overlay"}
		}
	}

	return &types.Project{
		Name:     name,
		Services: types.Services{name: composeSvc},
		Networks: projectNetworks,
		Volumes:  projectVolumes,
	}, nil
}

// ─── 辅助函数 ───

func swarmUpdateConfigToCompose(config *swarm.UpdateConfig) *types.UpdateConfig {
	if config == nil {
		return nil
	}
	parallelism := config.Parallelism
	return &types.UpdateConfig{
		Parallelism:     &parallelism,
		Delay:           types.Duration(config.Delay),
		FailureAction:   config.FailureAction,
		Monitor:         types.Duration(config.Monitor),
		MaxFailureRatio: config.MaxFailureRatio,
		Order:           config.Order,
	}
}

func swarmMountSource(m mount.Mount, containerDir string) string {
	switch m.Type {
	case mount.TypeBind:
		return relativeBindSource(m.Source, containerDir)
	case mount.TypeVolume:
		// swarm Mount.Source 对 named volume 即为卷名
		return m.Source
	}
	return m.Source
}

func swarmExtraHostsToMap(hosts []string) types.HostsList {
	if len(hosts) == 0 {
		return nil
	}
	result := types.HostsList{}
	for _, host := range hosts {
		fields := strings.Fields(host)
		if len(fields) >= 2 && fields[0] != "" && fields[1] != "" {
			result[fields[1]] = append(result[fields[1]], fields[0])
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
