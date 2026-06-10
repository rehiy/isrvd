package compose

import (
	"fmt"
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

	name := defaultString(spec.Name, svc.ID)
	composeSvc := types.ServiceConfig{
		Name:        name,
		Image:       cs.Image,
		Environment: sliceToEnv(cs.Env),
		Entrypoint:  types.ShellCommand(cs.Command),
		Command:     types.ShellCommand(cs.Args),
		WorkingDir:  cs.Dir,
		User:        cs.User,
		Hostname:    cs.Hostname,
		ExtraHosts:  swarmExtraHostsToMap(cs.Hosts),
		Tty:         cs.TTY,
		StdinOpen:   cs.OpenStdin,
		ReadOnly:    cs.ReadOnly,
		StopSignal:  cs.StopSignal,
		Sysctls:     types.Mapping(cs.Sysctls),
		CapAdd:      cs.CapabilityAdd,
		CapDrop:     cs.CapabilityDrop,
		Labels:      spec.Labels,
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
	if spec.TaskTemplate.Placement != nil && len(spec.TaskTemplate.Placement.Constraints) > 0 {
		if composeSvc.Deploy == nil {
			composeSvc.Deploy = &types.DeployConfig{}
		}
		composeSvc.Deploy.Placement.Constraints = spec.TaskTemplate.Placement.Constraints
	}

	if spec.EndpointSpec != nil {
		for _, p := range spec.EndpointSpec.Ports {
			composeSvc.Ports = append(composeSvc.Ports, types.ServicePortConfig{
				Target:    p.TargetPort,
				Published: strconv.Itoa(int(p.PublishedPort)),
				Protocol:  defaultString(string(p.Protocol), "tcp"),
				Mode:      defaultString(string(p.PublishMode), "ingress"),
			})
		}
	}

	for _, m := range cs.Mounts {
		composeSvc.Volumes = append(composeSvc.Volumes, types.ServiceVolumeConfig{
			Type:     string(m.Type),
			Source:   swarmMountSource(m, containerDir),
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
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
	}, nil
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
