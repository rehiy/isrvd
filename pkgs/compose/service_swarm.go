package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/pkgs/swarm"
)

// ServiceToSwarmRequest 将 compose ServiceConfig 转为 swarm.ServiceSpec
// project 用于将 service.Networks 的 key 解析为顶层 networks.<key>.name 指定的真实 docker 网络名。
func ServiceToSwarmRequest(project *types.Project, svc types.ServiceConfig) (swarm.ServiceSpec, error) {
	if svc.Image == "" {
		return swarm.ServiceSpec{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := defaultString(svc.Name, svc.ContainerName)
	mode, replicas := swarmDeployMode(svc)
	req := swarm.ServiceSpec{
		Name:     name,
		Image:    svc.Image,
		Env:      envToSlice(svc.Environment),
		Args:     []string(svc.Command),
		Mode:     mode,
		Replicas: replicas,
		Labels:   buildServiceLabels(project, svc, nil),
	}

	// networks
	for k := range svc.Networks {
		req.Networks = append(req.Networks, resolveNetworkName(project, k))
	}

	// ports
	applySwarmRequestPorts(svc, &req)

	// volumes
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
		req.Mounts = append(req.Mounts, swarm.ServiceMount{
			Type:     mountType,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}

	return req, nil
}

func swarmDeployMode(svc types.ServiceConfig) (string, *uint64) {
	replicas := uint64(1)
	mode := "replicated"
	if svc.Deploy != nil {
		if svc.Deploy.Mode == "global" {
			return "global", nil
		}
		if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
			replicas = uint64(*svc.Deploy.Replicas)
		}
	}
	return mode, &replicas
}

func applySwarmRequestPorts(svc types.ServiceConfig, req *swarm.ServiceSpec) {
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
		proto := defaultString(strings.ToLower(p.Protocol), "tcp")
		req.Ports = append(req.Ports, swarm.ServicePort{
			PublishedPort: uint32(published),
			TargetPort:    p.Target,
			Protocol:      proto,
			PublishMode:   defaultString(p.Mode, "ingress"),
		})
	}
}
