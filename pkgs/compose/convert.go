package compose

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// ServiceToCreateRequest 将 compose ServiceConfig 转换为 docker.ContainerCreateRequest
//
// 只覆盖 ContainerCreateRequest 已有的字段，未覆盖的 compose 特性
// (healthcheck / ulimits / sysctls 等) 保持忽略，后续按需扩展即可。
// project 用于将 service.Networks 的 key 解析为顶层 networks.<key>.name 指定的真实 docker 网络名。
func ServiceToCreateRequest(project *types.Project, svc types.ServiceConfig) (docker.ContainerCreateRequest, error) {
	if svc.Image == "" {
		return docker.ContainerCreateRequest{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := svc.ContainerName
	if name == "" {
		name = svc.Name
	}

	req := docker.ContainerCreateRequest{
		Image:      svc.Image,
		Name:       name,
		Cmd:        []string(svc.Command),
		Env:        environmentToSlice(svc.Environment),
		Network:    svc.NetworkMode,
		Workdir:    svc.WorkingDir,
		User:       svc.User,
		Hostname:   svc.Hostname,
		Privileged: svc.Privileged,
		CapAdd:     svc.CapAdd,
		CapDrop:    svc.CapDrop,
		Restart:    svc.Restart,
	}
	if req.Restart == "" {
		req.Restart = "no"
	}

	// 网络：优先 network_mode，其次取 networks 映射的第一个 key（解析为真实 docker 网络名）
	if req.Network == "" {
		for k := range svc.Networks {
			req.Network = resolveNetworkName(project, k)
			break
		}
	}

	// 端口：types.ServicePortConfig → map[hostPort]containerPort
	if len(svc.Ports) > 0 {
		req.Ports = make(map[string]string, len(svc.Ports))
		for _, p := range svc.Ports {
			if p.Target == 0 {
				continue
			}
			host := p.Published
			if host == "" {
				host = strconv.Itoa(int(p.Target))
			}
			req.Ports[host] = strconv.Itoa(int(p.Target))
		}
	}

	// 卷映射：只处理 bind 和 volume，tmpfs/image 暂不支持
	for _, v := range svc.Volumes {
		if v.Source == "" || v.Target == "" {
			continue
		}
		req.Volumes = append(req.Volumes, docker.VolumeMapping{
			HostPath:      v.Source,
			ContainerPath: v.Target,
			ReadOnly:      v.ReadOnly,
		})
	}

	// 资源限制：优先读 deploy.resources.limits，其次读顶层 mem_limit/cpus
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		lim := svc.Deploy.Resources.Limits
		if lim.MemoryBytes > 0 {
			req.Memory = int64(lim.MemoryBytes) / (1024 * 1024)
		}
		if lim.NanoCPUs > 0 {
			req.Cpus = float64(lim.NanoCPUs)
		}
	}
	if req.Memory == 0 && svc.MemLimit > 0 {
		req.Memory = int64(svc.MemLimit) / (1024 * 1024)
	}
	if req.Cpus == 0 && svc.CPUS > 0 {
		req.Cpus = float64(svc.CPUS)
	}

	return req, nil
}

// ServiceToSwarmRequest 将 compose ServiceConfig 转为 swarm.SwarmCreateServiceRequest
//
// project 用于将 service.Networks 的 key 解析为顶层 networks.<key>.name 指定的真实 docker 网络名。
func ServiceToSwarmRequest(project *types.Project, svc types.ServiceConfig) (swarm.SwarmCreateServiceRequest, error) {
	if svc.Image == "" {
		return swarm.SwarmCreateServiceRequest{}, fmt.Errorf("service %q 缺少 image", svc.Name)
	}

	name := svc.Name
	if name == "" {
		name = svc.ContainerName
	}

	req := swarm.SwarmCreateServiceRequest{
		Name:     name,
		Image:    svc.Image,
		Env:      environmentToSlice(svc.Environment),
		Args:     []string(svc.Command),
		Mode:     "replicated",
		Replicas: 1,
	}

	// 部署模式
	if svc.Deploy != nil {
		if svc.Deploy.Mode == "global" {
			req.Mode = "global"
			req.Replicas = 0
		} else if svc.Deploy.Replicas != nil && *svc.Deploy.Replicas > 0 {
			req.Replicas = *svc.Deploy.Replicas
		}
	}

	// 网络：解析为真实 docker 网络名
	for k := range svc.Networks {
		req.Networks = append(req.Networks, resolveNetworkName(project, k))
	}

	// 端口
	for _, p := range svc.Ports {
		if p.Target == 0 {
			continue
		}
		published, _ := strconv.Atoi(p.Published)
		if published == 0 {
			published = int(p.Target)
		}
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		req.Ports = append(req.Ports, swarm.SwarmPortConfig{
			Published: published,
			Target:    int(p.Target),
			Protocol:  proto,
		})
	}

	// 挂载
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
		req.Mounts = append(req.Mounts, swarm.SwarmMount{
			Type:   mountType,
			Source: v.Source,
			Target: v.Target,
		})
	}

	return req, nil
}

// extractNetworks 提取 service 使用的所有网络（用于在部署前确保网络存在）
// 返回解析后的真实 docker 网络名（应用 project.Networks[key].Name 覆盖）。
func extractNetworks(project *types.Project, svc types.ServiceConfig) []string {
	set := map[string]struct{}{}
	if svc.NetworkMode != "" && !isBuiltinNetworkMode(svc.NetworkMode) {
		set[svc.NetworkMode] = struct{}{}
	}
	for k := range svc.Networks {
		set[resolveNetworkName(project, k)] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

// resolveNetworkName 将 compose networks 的 key 解析为真实 docker 网络名
// 优先使用顶层 networks.<key>.name，未设置则回退到 key
func resolveNetworkName(project *types.Project, key string) string {
	if project != nil {
		if n, ok := project.Networks[key]; ok && n.Name != "" {
			return n.Name
		}
	}
	return key
}

// isBuiltinNetworkMode 判断是否为 docker 内置网络模式（不需要创建）
func isBuiltinNetworkMode(mode string) bool {
	m := strings.ToLower(mode)
	switch m {
	case "bridge", "host", "none", "default":
		return true
	}
	return strings.HasPrefix(m, "container:") || strings.HasPrefix(m, "service:")
}

// environmentToSlice 将 compose MappingWithEquals 转为 KEY=VALUE 列表
// value 为 nil 表示"不显式赋值"，此处跳过
func environmentToSlice(env types.MappingWithEquals) []string {
	if len(env) == 0 {
		return nil
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		if v == nil {
			continue
		}
		result = append(result, k+"="+*v)
	}
	return result
}

// sliceToEnv 将 KEY=VALUE 列表转成 MappingWithEquals
func sliceToEnv(env []string) types.MappingWithEquals {
	if len(env) == 0 {
		return nil
	}
	result := make(types.MappingWithEquals, len(env))
	for _, e := range env {
		if e == "" {
			continue
		}
		if idx := strings.Index(e, "="); idx > 0 {
			v := e[idx+1:]
			result[e[:idx]] = &v
		} else {
			result[e] = nil
		}
	}
	return result
}

// parsePort 解析 "8080" 或 "8080/tcp" 为端口号
func parsePort(s string) uint32 {
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(s)
	if n < 0 {
		return 0
	}
	return uint32(n)
}

// defaultString 若 v 为空则返回 def
func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
