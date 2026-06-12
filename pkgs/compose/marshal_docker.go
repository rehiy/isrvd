package compose

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/moby/docker-image-spec/specs-go/v1"
)

// ==================== Docker inspect -> Compose ====================

// DockerImageConfigResolver 按镜像引用读取镜像 Dockerfile 默认配置。
type DockerImageConfigResolver func(context.Context, string) (*v1.DockerOCIImageConfig, error)

// DockerImageConfigFromInspect 根据容器 inspect 信息解析镜像默认配置。
// 优先使用容器创建时的镜像引用，失败后回退到 inspect 中的镜像 ID。
func DockerImageConfigFromInspect(ctx context.Context, info container.InspectResponse, resolve DockerImageConfigResolver) *v1.DockerOCIImageConfig {
	if resolve == nil {
		return nil
	}
	if info.Config != nil && info.Config.Image != "" {
		if cfg, err := resolve(ctx, info.Config.Image); err == nil {
			return cfg
		}
	}
	if info.Image != "" {
		if cfg, err := resolve(ctx, info.Image); err == nil {
			return cfg
		}
	}
	return nil
}

// DockerImageConfigMapFromInspects 为一组容器 inspect 结果构建镜像默认配置缓存。
func DockerImageConfigMapFromInspects(ctx context.Context, infos []container.InspectResponse, resolve DockerImageConfigResolver) map[string]*v1.DockerOCIImageConfig {
	configs := make(map[string]*v1.DockerOCIImageConfig, len(infos))
	for _, info := range infos {
		if info.Config == nil || info.Config.Image == "" {
			continue
		}
		if _, ok := configs[info.Config.Image]; ok {
			continue
		}
		if cfg := DockerImageConfigFromInspect(ctx, info, resolve); cfg != nil {
			configs[info.Config.Image] = cfg
			if info.Image != "" {
				configs[info.Image] = cfg
			}
		}
	}
	return configs
}

// DockerProjectYAMLFromInspect 将单个 Docker 容器 inspect 结果反推为 compose YAML。
func DockerProjectYAMLFromInspect(ctx context.Context, info container.InspectResponse, resolve DockerImageConfigResolver, containerDir string) (string, error) {
	imageConfig := DockerImageConfigFromInspect(ctx, info, resolve)
	project, err := ProjectFromDockerInspect(info, imageConfig, containerDir)
	if err != nil {
		return "", err
	}
	data, err := ProjectToYAML(project)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DockerProjectYAMLFromInspects 将同一 Compose 项目的多个 Docker 容器 inspect 结果反推为 compose YAML。
func DockerProjectYAMLFromInspects(ctx context.Context, infos []container.InspectResponse, resolve DockerImageConfigResolver, projectName, containerDir string) (string, error) {
	configs := DockerImageConfigMapFromInspects(ctx, infos, resolve)
	project, err := ProjectFromDockerInspects(infos, configs, projectName, containerDir)
	if err != nil {
		return "", err
	}
	data, err := ProjectToYAML(project)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ==================== Docker inspect -> Project ====================

// ProjectFromDockerInspect 将 docker inspect 结果反推为单服务 compose Project。
// containerDir 非空时，位于该目录内的 bind source 会输出为相对路径。
func ProjectFromDockerInspect(info container.InspectResponse, imageConfig *v1.DockerOCIImageConfig, containerDir string) (*types.Project, error) {
	containerName := dockerInspectName(info)
	svc, projectNetworks, err := serviceFromDockerInspect(info, imageConfig, containerDir, containerName, containerName)
	if err != nil {
		return nil, err
	}
	return &types.Project{
		Name:     containerName,
		Services: types.Services{containerName: svc},
		Networks: projectNetworks,
	}, nil
}

// ProjectFromDockerInspects 将同一 compose project 的多个容器反推为多服务 compose Project。
// containerDir 非空时，位于该目录内的 bind source 会输出为相对路径。
func ProjectFromDockerInspects(infos []container.InspectResponse, imageConfigs map[string]*v1.DockerOCIImageConfig, projectName, containerDir string) (*types.Project, error) {
	if projectName == "" {
		return nil, fmt.Errorf("compose project 名称为空")
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("compose project %s 未找到容器", projectName)
	}

	services := make(types.Services, len(infos))
	projectNetworks := types.Networks{}
	for _, info := range infos {
		if dockerComposeOneoff(info) {
			continue
		}
		containerName := dockerInspectName(info)
		serviceName := dockerComposeServiceName(info)
		if serviceName == "" {
			serviceName = containerName
		}
		if _, ok := services[serviceName]; ok {
			return nil, fmt.Errorf("compose 服务 %s 存在多个容器，暂不支持接管 scaled 服务", serviceName)
		}

		var imageConfig *v1.DockerOCIImageConfig
		if info.Config != nil && imageConfigs != nil {
			imageConfig = imageConfigs[info.Config.Image]
		}
		svc, networks, err := serviceFromDockerInspect(info, imageConfig, containerDir, serviceName, containerName)
		if err != nil {
			return nil, err
		}
		services[serviceName] = svc
		for k, v := range networks {
			projectNetworks[k] = v
		}
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("compose project %s 未找到可接管服务容器", projectName)
	}

	return &types.Project{
		Name:     projectName,
		Services: services,
		Networks: projectNetworks,
	}, nil
}

func serviceFromDockerInspect(info container.InspectResponse, imageConfig *v1.DockerOCIImageConfig, containerDir, serviceName, containerName string) (types.ServiceConfig, types.Networks, error) {
	if info.Config == nil || info.HostConfig == nil {
		return types.ServiceConfig{}, nil, fmt.Errorf("容器 inspect 数据不完整")
	}

	var entrypoint types.ShellCommand
	if len(info.Config.Entrypoint) > 0 && (imageConfig == nil || !sliceEqual(info.Config.Entrypoint, imageConfig.Entrypoint)) {
		entrypoint = types.ShellCommand(info.Config.Entrypoint)
	}

	hostname := info.Config.Hostname
	if hostname == containerName || isContainerID(hostname) {
		hostname = ""
	}

	svc := types.ServiceConfig{
		Name:              serviceName,
		Image:             info.Config.Image,
		ContainerName:     containerName,
		Command:           types.ShellCommand(diffCmd(info.Config.Cmd, imageConfig)),
		Entrypoint:        entrypoint,
		Environment:       sliceToEnv(diffEnv(info.Config.Env, imageConfig)),
		DomainName:        info.Config.Domainname,
		WorkingDir:        diffString(info.Config.WorkingDir, imageConfig, func(c *v1.DockerOCIImageConfig) string { return c.WorkingDir }),
		User:              diffString(info.Config.User, imageConfig, func(c *v1.DockerOCIImageConfig) string { return c.User }),
		Hostname:          hostname,
		Privileged:        info.HostConfig.Privileged,
		CgroupParent:      info.HostConfig.CgroupParent,
		Cgroup:            diffDefaultString(string(info.HostConfig.CgroupnsMode), "private"),
		CapAdd:            []string(info.HostConfig.CapAdd),
		CapDrop:           []string(info.HostConfig.CapDrop),
		Restart:           restartPolicy(string(info.HostConfig.RestartPolicy.Name)),
		ExtraHosts:        extraHostsToMap(info.HostConfig.ExtraHosts),
		GroupAdd:          info.HostConfig.GroupAdd,
		Init:              info.HostConfig.Init,
		Ipc:               diffDefaultString(string(info.HostConfig.IpcMode), "private"),
		Isolation:         string(info.HostConfig.Isolation),
		OomKillDisable:    boolValue(info.HostConfig.OomKillDisable),
		Pid:               string(info.HostConfig.PidMode),
		PidsLimit:         int64Value(info.HostConfig.PidsLimit),
		Runtime:           diffDefaultString(info.HostConfig.Runtime, "runc"),
		SecurityOpt:       info.HostConfig.SecurityOpt,
		ShmSize:           diffDefaultShmSize(info.HostConfig.ShmSize),
		Tmpfs:             tmpfsToList(info.HostConfig.Tmpfs),
		Ulimits:           inspectUlimits(info.HostConfig.Ulimits),
		Uts:               string(info.HostConfig.UTSMode),
		Tty:               info.Config.Tty,
		StdinOpen:         info.Config.OpenStdin,
		ReadOnly:          info.HostConfig.ReadonlyRootfs,
		StopSignal:        diffString(info.Config.StopSignal, imageConfig, func(c *v1.DockerOCIImageConfig) string { return c.StopSignal }),
		Sysctls:           types.Mapping(info.HostConfig.Sysctls),
		DeviceCgroupRules: info.HostConfig.DeviceCgroupRules,
		Devices:           inspectDeviceMappings(info.HostConfig.Devices),
		Labels:            diffLabels(info.Config.Labels, imageConfig),
	}

	applyInspectDNS(&svc, info)
	projectNetworks := applyInspectNetworks(&svc, info, containerName)
	applyInspectPorts(&svc, info, imageConfig)
	applyInspectVolumes(&svc, info, containerDir)
	applyInspectResources(&svc, info)
	applyInspectDeviceRequests(&svc, info)

	return svc, projectNetworks, nil
}

func dockerInspectName(info container.InspectResponse) string {
	return defaultString(strings.TrimPrefix(info.Name, "/"), info.ID)
}

func dockerComposeServiceName(info container.InspectResponse) string {
	if info.Config == nil || info.Config.Labels == nil {
		return ""
	}
	return info.Config.Labels[ComposeServiceLabel]
}

func dockerComposeOneoff(info container.InspectResponse) bool {
	if info.Config == nil || info.Config.Labels == nil {
		return false
	}
	return strings.EqualFold(info.Config.Labels[ComposeOneoffLabel], "true")
}

// ==================== Docker inspect field mapping ====================

func applyInspectDNS(svc *types.ServiceConfig, info container.InspectResponse) {
	if len(info.HostConfig.DNS) > 0 {
		svc.DNS = info.HostConfig.DNS
	}
	if len(info.HostConfig.DNSSearch) > 0 {
		svc.DNSSearch = info.HostConfig.DNSSearch
	}
}

func applyInspectNetworks(svc *types.ServiceConfig, info container.InspectResponse, name string) types.Networks {
	networkMode := string(info.HostConfig.NetworkMode)
	// 内置模式（bridge/host/none/container:*/service:*）直接写入 network_mode，不解析 NetworkSettings
	if isBuiltinNetworkMode(networkMode) {
		svc.NetworkMode = networkMode
		return nil
	}
	// networkMode 为空或自定义网络名时，优先从 NetworkSettings.Networks 读取实际连接的网络
	if info.NetworkSettings == nil || len(info.NetworkSettings.Networks) == 0 {
		if networkMode != "" {
			svc.NetworkMode = networkMode
		}
		return nil
	}

	svc.Networks = make(map[string]*types.ServiceNetworkConfig, len(info.NetworkSettings.Networks))
	projectNetworks := make(types.Networks, len(info.NetworkSettings.Networks))
	for netName, ep := range info.NetworkSettings.Networks {
		netCfg := &types.ServiceNetworkConfig{}
		if ep.IPAMConfig != nil {
			netCfg.Ipv4Address = ep.IPAMConfig.IPv4Address
			netCfg.Ipv6Address = ep.IPAMConfig.IPv6Address
		}
		for _, alias := range ep.Aliases {
			if alias != name && !isContainerID(alias) {
				netCfg.Aliases = append(netCfg.Aliases, alias)
			}
		}
		svc.Networks[netName] = netCfg
		projectNetworks[netName] = types.NetworkConfig{External: true}
	}
	return projectNetworks
}

func applyInspectPorts(svc *types.ServiceConfig, info container.InspectResponse, imageConfig *v1.DockerOCIImageConfig) {
	seen := map[string]struct{}{}
	for containerPort, bindings := range info.HostConfig.PortBindings {
		target := parsePort(containerPort.Port())
		proto := defaultString(containerPort.Proto(), "tcp")
		if len(bindings) == 0 {
			continue
		}
		for _, b := range bindings {
			seen[string(containerPort)] = struct{}{}
			svc.Ports = append(svc.Ports, types.ServicePortConfig{
				Target:    target,
				Published: b.HostPort,
				HostIP:    b.HostIP,
				Protocol:  proto,
				Mode:      "ingress",
			})
		}
	}
	for containerPort := range info.Config.ExposedPorts {
		if _, ok := seen[string(containerPort)]; ok {
			continue
		}
		if imageHasExposedPort(imageConfig, string(containerPort)) {
			continue
		}
		svc.Expose = append(svc.Expose, strconv.Itoa(containerPort.Int()))
	}
}

func imageHasExposedPort(imageConfig *v1.DockerOCIImageConfig, port string) bool {
	if imageConfig == nil || len(imageConfig.ExposedPorts) == 0 {
		return false
	}
	_, ok := imageConfig.ExposedPorts[port]
	return ok
}

func applyInspectVolumes(svc *types.ServiceConfig, info container.InspectResponse, containerDir string) {
	for _, m := range info.Mounts {
		if m.Destination == "" {
			continue
		}
		// docker mount.Type 值（"bind"/"volume"/"tmpfs"）与 compose VolumeType 字符串一致
		svc.Volumes = append(svc.Volumes, types.ServiceVolumeConfig{
			Type:     string(m.Type),
			Source:   inspectMountSource(m, containerDir),
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}
	if len(svc.Volumes) > 0 {
		return
	}
	for _, bind := range info.HostConfig.Binds {
		parts := strings.SplitN(bind, ":", 3)
		if len(parts) < 2 {
			continue
		}
		vol := types.ServiceVolumeConfig{
			Type:   types.VolumeTypeBind,
			Source: relativeBindSource(parts[0], containerDir),
			Target: parts[1],
		}
		if len(parts) == 3 && strings.Contains(parts[2], "ro") {
			vol.ReadOnly = true
		}
		svc.Volumes = append(svc.Volumes, vol)
	}
}

func inspectMountSource(m container.MountPoint, containerDir string) string {
	switch m.Type {
	case mount.TypeBind:
		return relativeBindSource(m.Source, containerDir)
	case mount.TypeVolume:
		if m.Name != "" {
			return m.Name
		}
	}
	return m.Source
}

func relativeBindSource(source, baseDir string) string {
	if source == "" || baseDir == "" || !filepath.IsAbs(source) {
		return source
	}

	rel, err := filepath.Rel(filepath.Clean(baseDir), filepath.Clean(source))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return source
	}
	// source == baseDir 本身时保留绝对路径，避免挂载整个容器目录
	if rel == "." {
		return source
	}
	return "." + string(filepath.Separator) + rel
}

func applyInspectResources(svc *types.ServiceConfig, info container.InspectResponse) {
	if info.HostConfig.Memory == 0 && info.HostConfig.NanoCPUs == 0 {
		return
	}
	ensureDeploy(svc)
	svc.Deploy.Resources.Limits = &types.Resource{
		MemoryBytes: types.UnitBytes(info.HostConfig.Memory),
		NanoCPUs:    types.NanoCPUs(float64(info.HostConfig.NanoCPUs) / 1e9),
	}
}

// ==================== Docker inspect value converters ====================

func boolValue(v *bool) bool {
	return v != nil && *v
}

func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func tmpfsToList(tmpfs map[string]string) types.StringList {
	if len(tmpfs) == 0 {
		return nil
	}
	result := make(types.StringList, 0, len(tmpfs))
	for target, options := range tmpfs {
		if options == "" {
			result = append(result, target)
		} else {
			result = append(result, target+":"+options)
		}
	}
	return result
}

func inspectUlimits(ulimits []*container.Ulimit) map[string]*types.UlimitsConfig {
	if len(ulimits) == 0 {
		return nil
	}
	result := make(map[string]*types.UlimitsConfig, len(ulimits))
	for _, u := range ulimits {
		if u == nil || u.Name == "" {
			continue
		}
		if u.Soft == u.Hard {
			result[u.Name] = &types.UlimitsConfig{Single: int(u.Soft)}
		} else {
			result[u.Name] = &types.UlimitsConfig{Soft: int(u.Soft), Hard: int(u.Hard)}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func inspectDeviceMappings(devices []container.DeviceMapping) []types.DeviceMapping {
	if len(devices) == 0 {
		return nil
	}
	result := make([]types.DeviceMapping, 0, len(devices))
	for _, dev := range devices {
		result = append(result, types.DeviceMapping{
			Source:      dev.PathOnHost,
			Target:      dev.PathInContainer,
			Permissions: dev.CgroupPermissions,
		})
	}
	return result
}

func applyInspectDeviceRequests(svc *types.ServiceConfig, info container.InspectResponse) {
	if len(info.HostConfig.DeviceRequests) == 0 {
		return
	}
	ensureDeploy(svc)
	if svc.Deploy.Resources.Reservations == nil {
		svc.Deploy.Resources.Reservations = &types.Resource{}
	}
	for _, req := range info.HostConfig.DeviceRequests {
		var caps []string
		if len(req.Capabilities) > 0 {
			caps = req.Capabilities[0]
		}
		svc.Deploy.Resources.Reservations.Devices = append(svc.Deploy.Resources.Reservations.Devices, types.DeviceRequest{
			Driver:       req.Driver,
			Count:        types.DeviceCount(req.Count),
			IDs:          req.DeviceIDs,
			Capabilities: caps,
			Options:      types.Mapping(req.Options),
		})
	}
}

func ensureDeploy(svc *types.ServiceConfig) {
	if svc.Deploy == nil {
		svc.Deploy = &types.DeployConfig{}
	}
}

// ==================== Dockerfile default diff helpers ====================

// diffCmd 若容器 CMD 与镜像默认 CMD 相同则返回 nil（不写入 compose）
func diffCmd(containerCmd []string, imageConfig *v1.DockerOCIImageConfig) []string {
	if imageConfig == nil {
		return containerCmd
	}
	if sliceEqual(containerCmd, imageConfig.Cmd) {
		return nil
	}
	return containerCmd
}

// diffEnv 过滤掉镜像默认 ENV，只保留容器中新增或覆盖的环境变量
func diffEnv(containerEnv []string, imageConfig *v1.DockerOCIImageConfig) []string {
	if imageConfig == nil {
		return containerEnv
	}
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

// diffLabels 过滤掉镜像默认 Labels（Dockerfile LABEL），只保留容器层新增或覆盖的标签
func diffLabels(containerLabels map[string]string, imageConfig *v1.DockerOCIImageConfig) map[string]string {
	if len(containerLabels) == 0 {
		return nil
	}
	var result map[string]string
	for k, v := range containerLabels {
		if ignoreGeneratedDockerLabel(k) {
			continue
		}
		if imageConfig != nil && imageConfig.Labels != nil {
			if imgV, ok := imageConfig.Labels[k]; ok && imgV == v {
				continue // 与镜像默认值相同，跳过
			}
		}
		if result == nil {
			result = make(map[string]string)
		}
		result[k] = v
	}
	return result
}

func ignoreGeneratedDockerLabel(key string) bool {
	return strings.HasPrefix(key, "com.docker.compose.") || strings.HasPrefix(key, "com.docker.swarm.")
}

// diffString 若容器字段值与镜像默认值相同则返回空字符串（不写入 compose）
func diffString(containerVal string, imageConfig *v1.DockerOCIImageConfig, getter func(*v1.DockerOCIImageConfig) string) string {
	if imageConfig == nil || containerVal == "" {
		return containerVal
	}
	if containerVal == getter(imageConfig) {
		return ""
	}
	return containerVal
}

// ==================== Small conversion helpers ====================

// extraHostsToMap 将 []string{"host:ip"} 转换为 compose HostsList（map[string][]string）
func extraHostsToMap(hosts []string) types.HostsList {
	if len(hosts) == 0 {
		return nil
	}
	result := types.HostsList{}
	for _, host := range hosts {
		parts := strings.SplitN(host, ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			result[parts[0]] = append(result[parts[0]], parts[1])
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// isContainerID 判断字符串是否为 Docker 容器 ID（12 或 64 位十六进制）
func isContainerID(s string) bool {
	if len(s) != 12 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// restartPolicy 将 Docker RestartPolicy.Name 映射到 compose restart 值。
// Docker 的 "" 和 "no" 均表示不重启，compose 对应 "no"；
// 其他值（always/on-failure/unless-stopped）直接透传。
func restartPolicy(name string) string {
	if name == "" || name == "no" {
		return "no"
	}
	return name
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

// diffDefaultString 若值等于 Docker 默认值则返回空字符串，避免将默认行为写入 compose
func diffDefaultString(val, dockerDefault string) string {
	if val == dockerDefault {
		return ""
	}
	return val
}

// diffDefaultShmSize 若 shm_size 等于 Docker 默认值（64MB）则返回 0，不写入 compose
func diffDefaultShmSize(size int64) types.UnitBytes {
	const dockerDefaultShmSize int64 = 67108864 // 64MB
	if size == dockerDefaultShmSize {
		return 0
	}
	return types.UnitBytes(size)
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
