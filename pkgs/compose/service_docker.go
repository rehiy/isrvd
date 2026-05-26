package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/pkgs/docker"
)

// ServiceToDockerRequest 将 compose ServiceConfig 转换为 docker.ContainerSpec
// 只覆盖 ContainerSpec 已有的字段，未覆盖的 compose 特性
// (healthcheck / ulimits / sysctls 等) 保持忽略，后续按需扩展即可。
// project 用于将 service.Networks 的 key 解析为顶层 networks.<key>.name 指定的真实 docker 网络名。
func ServiceToDockerRequest(project *types.Project, svc types.ServiceConfig) (docker.ContainerSpec, error) {
	if svc.Image == "" {
		return docker.ContainerSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := defaultString(svc.ContainerName, svc.Name)
	req := docker.ContainerSpec{
		Image:      svc.Image,
		Name:       name,
		Cmd:        []string(svc.Command),
		Env:        envToSlice(svc.Environment),
		Network:    svc.NetworkMode,
		Workdir:    svc.WorkingDir,
		User:       svc.User,
		Hostname:   svc.Hostname,
		Privileged: svc.Privileged,
		CapAdd:     svc.CapAdd,
		CapDrop:    svc.CapDrop,
		Restart:    defaultString(svc.Restart, "always"), // compose 未指定时默认 always（与 restartPolicy 方向相反：inspect→compose 映射空串为"no"）
		Labels: buildServiceLabels(project, svc, map[string]string{
			ComposeContainerNumberLabel: "1",
			ComposeOneoffLabel:          "False",
		}),
	}

	applyDockerNetwork(project, svc, &req)
	applyDockerPorts(svc, &req)
	applyDockerVolumes(svc, &req)
	applyDockerResources(svc, &req)

	return req, nil
}

func applyDockerNetwork(project *types.Project, svc types.ServiceConfig, req *docker.ContainerSpec) {
	if req.Network != "" {
		return
	}
	for k := range svc.Networks {
		req.Network = resolveNetworkName(project, k)
		break
	}
}

func applyDockerPorts(svc types.ServiceConfig, req *docker.ContainerSpec) {
	if len(svc.Ports) == 0 {
		return
	}
	req.Ports = make(map[string]string, len(svc.Ports))
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		proto := defaultString(strings.ToLower(p.Protocol), "tcp")
		host := p.Published
		if host == "" {
			host = strconv.Itoa(int(p.Target))
		}
		if p.HostIP != "" && p.HostIP != "0.0.0.0" {
			host = p.HostIP + ":" + host
		}
		req.Ports[host+"/"+proto] = strconv.Itoa(int(p.Target))
	}
}

func applyDockerVolumes(svc types.ServiceConfig, req *docker.ContainerSpec) {
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
		req.Volumes = append(req.Volumes, docker.VolumeMapping{
			Type:          string(mountType),
			Source:        v.Source,
			ContainerPath: v.Target,
			ReadOnly:      v.ReadOnly,
		})
	}
}

func applyDockerResources(svc types.ServiceConfig, req *docker.ContainerSpec) {
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		lim := svc.Deploy.Resources.Limits
		if lim.MemoryBytes > 0 {
			req.Memory = int64(lim.MemoryBytes) / (1024 * 1024)
		}
		if lim.NanoCPUs > 0 {
			req.Cpus = float64(lim.NanoCPUs) / 1e9
		}
	}
	if req.Memory == 0 && svc.MemLimit > 0 {
		req.Memory = int64(svc.MemLimit) / (1024 * 1024)
	}
	if req.Cpus == 0 && svc.CPUS > 0 {
		req.Cpus = float64(svc.CPUS)
	}
}
