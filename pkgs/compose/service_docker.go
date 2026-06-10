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
	"github.com/docker/go-units"
)

// ==================== Compose service -> Docker create spec ====================

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
			Domainname: svc.DomainName,
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
			CgroupnsMode:   container.CgroupnsMode(svc.Cgroup),
			DNS:            []string(svc.DNS),
			DNSOptions:     svc.DNSOpts,
			DNSSearch:      []string(svc.DNSSearch),
			ExtraHosts:     svc.ExtraHosts.AsList(":"),
			GroupAdd:       svc.GroupAdd,
			Init:           svc.Init,
			IpcMode:        container.IpcMode(svc.Ipc),
			Isolation:      container.Isolation(svc.Isolation),
			PidMode:        container.PidMode(svc.Pid),
			ReadonlyRootfs: svc.ReadOnly,
			RestartPolicy:  container.RestartPolicy{Name: container.RestartPolicyMode(restartPolicyName(svc.Restart))},
			Resources: container.Resources{
				CgroupParent:      svc.CgroupParent,
				DeviceCgroupRules: svc.DeviceCgroupRules,
				Devices:           dockerDeviceMappings(svc.Devices),
				OomKillDisable:    boolPtr(svc.OomKillDisable),
				PidsLimit:         int64Ptr(svc.PidsLimit),
				Ulimits:           dockerUlimits(svc.Ulimits),
			},
			Runtime:     svc.Runtime,
			SecurityOpt: svc.SecurityOpt,
			ShmSize:     int64(svc.ShmSize),
			Sysctls:     map[string]string(svc.Sysctls),
			Tmpfs:       tmpfsMap(svc.Tmpfs),
			UTSMode:     container.UTSMode(svc.Uts),
		},
	}

	applyDockerNetwork(project, svc, spec.HostConfig)
	applyDockerExpose(svc, spec.Config)
	applyDockerPorts(svc, spec.Config, spec.HostConfig)
	applyDockerVolumes(svc, spec.HostConfig)
	applyDockerResources(svc, spec.HostConfig)
	applyDockerDeviceRequests(svc, spec.HostConfig)

	return spec, nil
}

// ==================== Docker host/config mapping ====================

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

func applyDockerExpose(svc types.ServiceConfig, containerConfig *container.Config) {
	if len(svc.Expose) == 0 {
		return
	}
	if containerConfig.ExposedPorts == nil {
		containerConfig.ExposedPorts = make(nat.PortSet)
	}
	for _, raw := range svc.Expose {
		proto := "tcp"
		port := raw
		if idx := strings.LastIndex(raw, "/"); idx >= 0 {
			port = raw[:idx]
			proto = raw[idx+1:]
		}
		containerConfig.ExposedPorts[nat.Port(port+"/"+proto)] = struct{}{}
	}
}

func applyDockerPorts(svc types.ServiceConfig, containerConfig *container.Config, hostConfig *container.HostConfig) {
	if len(svc.Ports) == 0 {
		return
	}
	if containerConfig.ExposedPorts == nil {
		containerConfig.ExposedPorts = make(nat.PortSet)
	}
	portBindings := make(nat.PortMap)
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		proto := defaultString(strings.ToLower(p.Protocol), "tcp")
		host := p.Published
		hostIP := "0.0.0.0"
		if p.HostIP != "" && p.HostIP != "0.0.0.0" {
			hostIP = p.HostIP
		}
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			hostIP = host[:idx]
			host = host[idx+1:]
		}
		port := nat.Port(strconv.Itoa(int(p.Target)) + "/" + proto)
		portBindings[port] = append(portBindings[port], nat.PortBinding{HostIP: hostIP, HostPort: host})
		containerConfig.ExposedPorts[port] = struct{}{}
	}
	if len(portBindings) > 0 {
		hostConfig.PortBindings = portBindings
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

// ==================== Docker value converters ====================

func boolPtr(v bool) *bool {
	if !v {
		return nil
	}
	return &v
}

func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func tmpfsMap(tmpfs types.StringList) map[string]string {
	if len(tmpfs) == 0 {
		return nil
	}
	result := make(map[string]string, len(tmpfs))
	for _, raw := range tmpfs {
		target, options, _ := strings.Cut(raw, ":")
		if target == "" {
			continue
		}
		result[target] = options
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func dockerUlimits(ulimits map[string]*types.UlimitsConfig) []*container.Ulimit {
	if len(ulimits) == 0 {
		return nil
	}
	result := make([]*container.Ulimit, 0, len(ulimits))
	for name, cfg := range ulimits {
		if cfg == nil || name == "" {
			continue
		}
		soft := int64(cfg.Soft)
		hard := int64(cfg.Hard)
		if cfg.Single != 0 {
			soft = int64(cfg.Single)
			hard = int64(cfg.Single)
		}
		result = append(result, &units.Ulimit{Name: name, Soft: soft, Hard: hard})
	}
	return result
}

// ==================== Docker device/GPU mapping ====================

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
	for _, dev := range svc.Gpus {
		hostConfig.DeviceRequests = append(hostConfig.DeviceRequests, dockerDeviceRequest(dev))
	}
	if svc.Deploy == nil || svc.Deploy.Resources.Reservations == nil {
		return
	}
	for _, dev := range svc.Deploy.Resources.Reservations.Devices {
		hostConfig.DeviceRequests = append(hostConfig.DeviceRequests, dockerDeviceRequest(dev))
	}
}

func dockerDeviceRequest(dev types.DeviceRequest) container.DeviceRequest {
	caps := dev.Capabilities
	if len(caps) == 0 {
		caps = []string{"gpu"}
	}
	return container.DeviceRequest{
		Driver:       dev.Driver,
		Count:        int(dev.Count),
		DeviceIDs:    dev.IDs,
		Capabilities: [][]string{caps},
		Options:      stringMap(dev.Options),
	}
}

func stringMap(values types.Mapping) map[string]string {
	if len(values) == 0 {
		return nil
	}
	return map[string]string(values)
}
