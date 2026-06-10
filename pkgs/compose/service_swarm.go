package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/mount"
	dockerswarm "github.com/docker/docker/api/types/swarm"
)

// ServiceToSwarmSpec 将 compose ServiceConfig 转为 Docker SDK Swarm ServiceSpec。
func ServiceToSwarmSpec(project *types.Project, svc types.ServiceConfig) (dockerswarm.ServiceSpec, error) {
	if svc.Image == "" {
		return dockerswarm.ServiceSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := defaultString(svc.Name, svc.ContainerName)
	spec := dockerswarm.ServiceSpec{
		Annotations: dockerswarm.Annotations{
			Name:   name,
			Labels: buildServiceLabels(project, svc, nil),
		},
		TaskTemplate: dockerswarm.TaskSpec{
			ContainerSpec: &dockerswarm.ContainerSpec{
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
		EndpointSpec: &dockerswarm.EndpointSpec{},
	}

	if len(svc.DNS) > 0 || len(svc.DNSOpts) > 0 || len(svc.DNSSearch) > 0 {
		spec.TaskTemplate.ContainerSpec.DNSConfig = &dockerswarm.DNSConfig{
			Nameservers: []string(svc.DNS),
			Options:     svc.DNSOpts,
			Search:      []string(svc.DNSSearch),
		}
	}

	applySwarmMode(svc, &spec)
	applySwarmNetworks(project, svc, &spec)
	applySwarmPorts(svc, &spec)
	applySwarmVolumes(svc, &spec)
	applySwarmPlacement(svc, &spec)
	applySwarmResources(svc, &spec)

	return spec, nil
}

func applySwarmMode(svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	replicas := uint64(1)
	if svc.Deploy != nil {
		if svc.Deploy.Mode == "global" {
			spec.Mode = dockerswarm.ServiceMode{Global: &dockerswarm.GlobalService{}}
			return
		}
		if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
			replicas = uint64(*svc.Deploy.Replicas)
		}
	}
	spec.Mode = dockerswarm.ServiceMode{Replicated: &dockerswarm.ReplicatedService{Replicas: &replicas}}
}

func applySwarmNetworks(project *types.Project, svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	for _, k := range svc.NetworksByPriority() {
		spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, dockerswarm.NetworkAttachmentConfig{Target: resolveNetworkName(project, k)})
	}
}

func applySwarmPorts(svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		publishedStr := p.Published
		if idx := strings.LastIndex(publishedStr, ":"); idx >= 0 {
			publishedStr = publishedStr[idx+1:]
		}
		published, _ := strconv.Atoi(publishedStr)
		if published == 0 {
			published = int(p.Target)
		}
		proto := dockerswarm.PortConfigProtocolTCP
		if strings.EqualFold(p.Protocol, "udp") {
			proto = dockerswarm.PortConfigProtocolUDP
		}
		publishMode := dockerswarm.PortConfigPublishModeIngress
		if strings.EqualFold(p.Mode, "host") {
			publishMode = dockerswarm.PortConfigPublishModeHost
		}
		spec.EndpointSpec.Ports = append(spec.EndpointSpec.Ports, dockerswarm.PortConfig{
			PublishedPort: uint32(published),
			TargetPort:    p.Target,
			Protocol:      proto,
			PublishMode:   publishMode,
		})
	}
}

func applySwarmVolumes(svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	for _, v := range svc.Volumes {
		if v.Source == "" || v.Target == "" {
			continue
		}
		mountType := v.Type
		if mountType == "" {
			if strings.HasPrefix(v.Source, "/") || strings.HasPrefix(v.Source, ".") {
				mountType = types.VolumeTypeBind
			} else {
				mountType = types.VolumeTypeVolume
			}
		}
		spec.TaskTemplate.ContainerSpec.Mounts = append(spec.TaskTemplate.ContainerSpec.Mounts, mount.Mount{
			Type:     mount.Type(mountType),
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
}

func applySwarmPlacement(svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	if svc.Deploy == nil || len(svc.Deploy.Placement.Constraints) == 0 {
		return
	}
	spec.TaskTemplate.Placement = &dockerswarm.Placement{Constraints: svc.Deploy.Placement.Constraints}
}

func applySwarmResources(svc types.ServiceConfig, spec *dockerswarm.ServiceSpec) {
	if svc.Deploy == nil {
		return
	}
	if limits := svc.Deploy.Resources.Limits; limits != nil {
		spec.TaskTemplate.Resources.Limits = &dockerswarm.Limit{
			MemoryBytes: int64(limits.MemoryBytes),
			Pids:        limits.Pids,
		}
		if limits.NanoCPUs > 0 {
			spec.TaskTemplate.Resources.Limits.NanoCPUs = int64(float64(limits.NanoCPUs) * 1e9)
		}
	}
	if reservations := svc.Deploy.Resources.Reservations; reservations != nil {
		spec.TaskTemplate.Resources.Reservations = &dockerswarm.Resources{
			MemoryBytes: int64(reservations.MemoryBytes),
		}
		if reservations.NanoCPUs > 0 {
			spec.TaskTemplate.Resources.Reservations.NanoCPUs = int64(float64(reservations.NanoCPUs) * 1e9)
		}
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
