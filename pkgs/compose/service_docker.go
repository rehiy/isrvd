package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// DockerCreateSpec 是 Docker SDK ContainerCreate 的参数集合。
type DockerCreateSpec struct {
	Name             string
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

// ServiceToDockerCreateSpec 将 compose ServiceConfig 转换为 Docker SDK 创建参数。
func ServiceToDockerCreateSpec(project *types.Project, svc types.ServiceConfig) (DockerCreateSpec, error) {
	if svc.Image == "" {
		return DockerCreateSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := defaultString(svc.ContainerName, svc.Name)
	spec := DockerCreateSpec{
		Name: name,
		Config: &container.Config{
			Image:      svc.Image,
			Cmd:        []string(svc.Command),
			Entrypoint: []string(svc.Entrypoint),
			Env:        envToSlice(svc.Environment),
			WorkingDir: svc.WorkingDir,
			User:       svc.User,
			Hostname:   svc.Hostname,
			Tty:        svc.Tty,
			OpenStdin:  svc.StdinOpen,
			StopSignal: svc.StopSignal,
			Labels: buildServiceLabels(project, svc, map[string]string{
				ComposeContainerNumberLabel: "1",
				ComposeOneoffLabel:          "False",
			}),
		},
		HostConfig: &container.HostConfig{
			Privileged:     svc.Privileged,
			CapAdd:         svc.CapAdd,
			CapDrop:        svc.CapDrop,
			DNS:            []string(svc.DNS),
			DNSOptions:     svc.DNSOpts,
			DNSSearch:      []string(svc.DNSSearch),
			ExtraHosts:     svc.ExtraHosts.AsList(":"),
			ReadonlyRootfs: svc.ReadOnly,
			Sysctls:        map[string]string(svc.Sysctls),
			RestartPolicy:  container.RestartPolicy{Name: container.RestartPolicyMode(restartPolicyName(svc.Restart))},
		},
	}

	spec.HostConfig.Devices = dockerDeviceMappings(svc.Devices)
	spec.HostConfig.DeviceCgroupRules = svc.DeviceCgroupRules

	applyDockerNetwork(project, svc, spec.HostConfig)
	applyDockerPorts(svc, spec.Config, spec.HostConfig)
	applyDockerVolumes(svc, spec.HostConfig)
	applyDockerResources(svc, spec.HostConfig)
	applyDockerDeviceRequests(svc, spec.HostConfig)

	return spec, nil
}

func restartPolicyName(name string) string {
	switch name {
	case "always", "on-failure", "unless-stopped":
		return name
	default:
		return "no"
	}
}

func applyDockerNetwork(project *types.Project, svc types.ServiceConfig, hostConfig *container.HostConfig) {
	if svc.NetworkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(svc.NetworkMode)
		return
	}
	for _, k := range svc.NetworksByPriority() {
		hostConfig.NetworkMode = container.NetworkMode(resolveNetworkName(project, k))
		break
	}
}

func applyDockerPorts(svc types.ServiceConfig, containerConfig *container.Config, hostConfig *container.HostConfig) {
	if len(svc.Ports) == 0 {
		return
	}
	portBindings := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		proto := defaultString(strings.ToLower(p.Protocol), "tcp")
		host := p.Published
		if host == "" {
			host = strconv.Itoa(int(p.Target))
		}
		hostIP := "0.0.0.0"
		if p.HostIP != "" && p.HostIP != "0.0.0.0" {
			hostIP = p.HostIP
		}
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			hostIP = host[:idx]
			host = host[idx+1:]
		}
		port := nat.Port(strconv.Itoa(int(p.Target)) + "/" + proto)
		portBindings[port] = []nat.PortBinding{{HostIP: hostIP, HostPort: host}}
		exposedPorts[port] = struct{}{}
	}
	if len(portBindings) > 0 {
		hostConfig.PortBindings = portBindings
		containerConfig.ExposedPorts = exposedPorts
	}
}

func applyDockerVolumes(svc types.ServiceConfig, hostConfig *container.HostConfig) {
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
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.Type(mountType),
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
}

func applyDockerResources(svc types.ServiceConfig, hostConfig *container.HostConfig) {
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		lim := svc.Deploy.Resources.Limits
		if lim.MemoryBytes > 0 {
			hostConfig.Memory = int64(lim.MemoryBytes)
		}
		if lim.NanoCPUs > 0 {
			hostConfig.NanoCPUs = int64(float64(lim.NanoCPUs) * 1e9)
		}
	}
	if hostConfig.Memory == 0 && svc.MemLimit > 0 {
		hostConfig.Memory = int64(svc.MemLimit)
	}
	if hostConfig.NanoCPUs == 0 && svc.CPUS > 0 {
		hostConfig.NanoCPUs = int64(float64(svc.CPUS) * 1e9)
	}
}

func dockerDeviceMappings(devices []types.DeviceMapping) []container.DeviceMapping {
	if len(devices) == 0 {
		return nil
	}
	result := make([]container.DeviceMapping, 0, len(devices))
	for _, dev := range devices {
		result = append(result, container.DeviceMapping{
			PathOnHost:        dev.Source,
			PathInContainer:   defaultString(dev.Target, dev.Source),
			CgroupPermissions: defaultString(dev.Permissions, "rwm"),
		})
	}
	return result
}

func applyDockerDeviceRequests(svc types.ServiceConfig, hostConfig *container.HostConfig) {
	if svc.Deploy == nil || svc.Deploy.Resources.Reservations == nil {
		return
	}
	for _, dev := range svc.Deploy.Resources.Reservations.Devices {
		caps := dev.Capabilities
		if len(caps) == 0 {
			caps = []string{"gpu"}
		}
		hostConfig.DeviceRequests = append(hostConfig.DeviceRequests, container.DeviceRequest{
			Driver:       dev.Driver,
			Count:        int(dev.Count),
			DeviceIDs:    dev.IDs,
			Capabilities: [][]string{caps},
		})
	}
}
