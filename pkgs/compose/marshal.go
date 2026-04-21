package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	dockertypes "github.com/docker/docker/api/types"

	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// ProjectFromCreateRequest 将 ContainerCreateRequest 转为单服务 compose Project
func ProjectFromCreateRequest(req docker.ContainerCreateRequest) (*types.Project, error) {
	if req.Name == "" || req.Image == "" {
		return nil, fmt.Errorf("name 或 image 为空")
	}

	svc := types.ServiceConfig{
		Name:          req.Name,
		Image:         req.Image,
		ContainerName: req.Name,
		Command:       types.ShellCommand(req.Cmd),
		Environment:   sliceToEnv(req.Env),
		NetworkMode:   req.Network,
		WorkingDir:    req.Workdir,
		User:          req.User,
		Hostname:      req.Hostname,
		Privileged:    req.Privileged,
		CapAdd:        req.CapAdd,
		CapDrop:       req.CapDrop,
		Restart:       defaultString(req.Restart, "always"),
	}

	for host, container := range req.Ports {
		svc.Ports = append(svc.Ports, types.ServicePortConfig{
			Target:    parsePort(container),
			Published: host,
			Protocol:  "tcp",
			Mode:      "ingress",
		})
	}

	for _, v := range req.Volumes {
		svc.Volumes = append(svc.Volumes, types.ServiceVolumeConfig{
			Type:     types.VolumeTypeBind,
			Source:   v.HostPath,
			Target:   v.ContainerPath,
			ReadOnly: v.ReadOnly,
		})
	}

	if req.Memory > 0 || req.Cpus > 0 {
		svc.Deploy = &types.DeployConfig{
			Resources: types.Resources{Limits: &types.Resource{
				MemoryBytes: types.UnitBytes(req.Memory * 1024 * 1024),
				NanoCPUs:    types.NanoCPUs(req.Cpus),
			}},
		}
	}

	return &types.Project{
		Name:     req.Name,
		Services: types.Services{req.Name: svc},
	}, nil
}

// ProjectFromInspect 将 docker inspect 结果反推为单服务 compose Project
func ProjectFromInspect(info dockertypes.ContainerJSON) (*types.Project, error) {
	if info.Config == nil || info.HostConfig == nil {
		return nil, fmt.Errorf("容器 inspect 数据不完整")
	}
	name := strings.TrimPrefix(info.Name, "/")
	if name == "" {
		name = info.ID
	}

	svc := types.ServiceConfig{
		Name:          name,
		Image:         info.Config.Image,
		ContainerName: name,
		Command:       types.ShellCommand(info.Config.Cmd),
		Environment:   sliceToEnv(info.Config.Env),
		WorkingDir:    info.Config.WorkingDir,
		User:          info.Config.User,
		Hostname:      info.Config.Hostname,
		Privileged:    info.HostConfig.Privileged,
		CapAdd:        []string(info.HostConfig.CapAdd),
		CapDrop:       []string(info.HostConfig.CapDrop),
		Restart:       defaultString(string(info.HostConfig.RestartPolicy.Name), "always"),
		NetworkMode:   string(info.HostConfig.NetworkMode),
	}

	// 端口映射：HostConfig.PortBindings (容器端口 → 宿主绑定列表)
	for containerPort, bindings := range info.HostConfig.PortBindings {
		target := parsePort(containerPort.Port())
		proto := containerPort.Proto()
		if proto == "" {
			proto = "tcp"
		}
		if len(bindings) == 0 {
			svc.Ports = append(svc.Ports, types.ServicePortConfig{
				Target:   target,
				Protocol: proto,
				Mode:     "ingress",
			})
			continue
		}
		for _, b := range bindings {
			svc.Ports = append(svc.Ports, types.ServicePortConfig{
				Target:    target,
				Published: b.HostPort,
				Protocol:  proto,
				Mode:      "ingress",
			})
		}
	}

	// 挂载：优先用 Mounts（结构化），其次 HostConfig.Binds（字符串回退）
	for _, m := range info.Mounts {
		if m.Destination == "" {
			continue
		}
		svc.Volumes = append(svc.Volumes, types.ServiceVolumeConfig{
			Type:     string(m.Type),
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}
	if len(svc.Volumes) == 0 {
		for _, bind := range info.HostConfig.Binds {
			parts := strings.SplitN(bind, ":", 3)
			if len(parts) < 2 {
				continue
			}
			vol := types.ServiceVolumeConfig{
				Type:   types.VolumeTypeBind,
				Source: parts[0],
				Target: parts[1],
			}
			if len(parts) == 3 && strings.Contains(parts[2], "ro") {
				vol.ReadOnly = true
			}
			svc.Volumes = append(svc.Volumes, vol)
		}
	}

	// 资源限制
	if info.HostConfig.Memory > 0 || info.HostConfig.NanoCPUs > 0 {
		svc.Deploy = &types.DeployConfig{
			Resources: types.Resources{Limits: &types.Resource{
				MemoryBytes: types.UnitBytes(info.HostConfig.Memory),
				NanoCPUs:    types.NanoCPUs(float64(info.HostConfig.NanoCPUs) / 1e9),
			}},
		}
	}

	return &types.Project{
		Name:     name,
		Services: types.Services{name: svc},
	}, nil
}

// ProjectFromSwarmInspect 将 SwarmServiceInspect 反推为单服务 compose Project
func ProjectFromSwarmInspect(info *swarm.SwarmServiceInspect) (*types.Project, error) {
	if info == nil || info.Image == "" {
		return nil, fmt.Errorf("swarm 服务数据不完整")
	}

	name := info.Name
	if name == "" {
		name = info.ID
	}

	svc := types.ServiceConfig{
		Name:        name,
		Image:       info.Image,
		Environment: sliceToEnv(info.Env),
		Command:     types.ShellCommand(info.Args),
	}

	// 部署模式
	if info.Mode == "global" {
		svc.Deploy = &types.DeployConfig{Mode: "global"}
	} else if info.Replicas != nil {
		r := int(*info.Replicas)
		svc.Deploy = &types.DeployConfig{
			Mode:     "replicated",
			Replicas: &r,
		}
	}

	// 端口映射
	for _, p := range info.Ports {
		svc.Ports = append(svc.Ports, types.ServicePortConfig{
			Target:    p.TargetPort,
			Published: strconv.Itoa(int(p.PublishedPort)),
			Protocol:  p.Protocol,
			Mode:      "ingress",
		})
	}

	// 挂载
	for _, m := range info.Mounts {
		svc.Volumes = append(svc.Volumes, types.ServiceVolumeConfig{
			Type:     m.Type,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// 网络
	if len(info.Networks) > 0 {
		svc.Networks = make(map[string]*types.ServiceNetworkConfig, len(info.Networks))
		for _, n := range info.Networks {
			svc.Networks[n] = &types.ServiceNetworkConfig{}
		}
	}

	return &types.Project{
		Name:     name,
		Services: types.Services{name: svc},
	}, nil
}

// ProjectToYAML 使用 compose-go 官方序列化器输出 YAML（带文件头）
func ProjectToYAML(project *types.Project) ([]byte, error) {
	if project == nil {
		return nil, fmt.Errorf("project 为空")
	}
	data, err := project.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("序列化 compose 失败: %w", err)
	}
	return append([]byte("# Auto-generated by isrvd\n\n"), data...), nil
}
