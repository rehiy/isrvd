package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

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

// ProjectFromInspect 将 docker inspect 结果反推为单服务 compose Project。
// imageConfig 为镜像的默认配置（来自 Dockerfile），用于过滤掉镜像内置的默认值，
// 避免将 Dockerfile 中的 CMD/ENV/WORKDIR/USER 等冗余写入 compose yml。
// imageConfig 可为 nil，此时不做过滤。
func ProjectFromInspect(info container.InspectResponse, imageConfig *dockerspec.DockerOCIImageConfig) (*types.Project, error) {
	if info.Config == nil || info.HostConfig == nil {
		return nil, fmt.Errorf("容器 inspect 数据不完整")
	}
	name := strings.TrimPrefix(info.Name, "/")
	if name == "" {
		name = info.ID
	}

	// 过滤 Dockerfile 默认值：只保留用户实际覆盖的配置
	cmd := diffCmd(info.Config.Cmd, imageConfig)
	env := diffEnv(info.Config.Env, imageConfig)
	workingDir := diffString(info.Config.WorkingDir, imageConfig, func(c *dockerspec.DockerOCIImageConfig) string { return c.WorkingDir })
	user := diffString(info.Config.User, imageConfig, func(c *dockerspec.DockerOCIImageConfig) string { return c.User })

	// hostname 默认为容器名（Docker 自动设置），不来自 Dockerfile，若相同则忽略
	hostname := info.Config.Hostname
	if hostname == name {
		hostname = ""
	}

	svc := types.ServiceConfig{
		Name:          name,
		Image:         info.Config.Image,
		ContainerName: name,
		Command:       types.ShellCommand(cmd),
		Environment:   sliceToEnv(env),
		WorkingDir:    workingDir,
		User:          user,
		Hostname:      hostname,
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

// ProjectFromSwarmInspect 将 ServiceInfo 反推为单服务 compose Project
func ProjectFromSwarmInspect(info *swarm.ServiceInfo) (*types.Project, error) {
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

// diffCmd 若容器 CMD 与镜像默认 CMD 相同则返回 nil（不写入 compose）
func diffCmd(containerCmd []string, imageConfig *dockerspec.DockerOCIImageConfig) []string {
	if imageConfig == nil {
		return containerCmd
	}
	if sliceEqual(containerCmd, imageConfig.Cmd) {
		return nil
	}
	return containerCmd
}

// diffEnv 过滤掉镜像默认 ENV，只保留容器中新增或覆盖的环境变量
func diffEnv(containerEnv []string, imageConfig *dockerspec.DockerOCIImageConfig) []string {
	if imageConfig == nil {
		return containerEnv
	}
	// 构建镜像默认 env 集合（KEY=VALUE 精确匹配）
	imageEnvSet := make(map[string]struct{}, len(imageConfig.Env))
	for _, e := range imageConfig.Env {
		imageEnvSet[e] = struct{}{}
	}
	var result []string
	for _, e := range containerEnv {
		if _, ok := imageEnvSet[e]; !ok {
			result = append(result, e)
		}
	}
	return result
}

// diffString 若容器字段值与镜像默认值相同则返回空字符串（不写入 compose）
func diffString(containerVal string, imageConfig *dockerspec.DockerOCIImageConfig, getter func(*dockerspec.DockerOCIImageConfig) string) string {
	if imageConfig == nil || containerVal == "" {
		return containerVal
	}
	if containerVal == getter(imageConfig) {
		return ""
	}
	return containerVal
}

// sliceEqual 判断两个字符串切片是否完全相同
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
