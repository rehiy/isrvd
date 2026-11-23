package compose

import (
	"maps"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

const (
	ComposeProjectLabel         = "com.docker.compose.project"
	ComposeServiceLabel         = "com.docker.compose.service"
	ComposeContainerNumberLabel = "com.docker.compose.container-number"
	ComposeOneoffLabel          = "com.docker.compose.oneoff"
	SwarmUpdateTokenLabel       = "io.isrvd.compose.update-token"
)

// ProjectNetworks 按实际名称保留项目网络的归属与驱动。具名 network_mode 引用现有网络。
func ProjectNetworks(project *types.Project) map[string]types.NetworkConfig {
	networks := make(map[string]types.NetworkConfig, len(project.Networks))
	for _, svc := range project.Services {
		for key := range svc.Networks {
			config := project.Networks[key]
			name := resolveNetworkName(project, key)
			config.External = config.External || networks[name].External
			networks[name] = config
		}
		if svc.NetworkMode != "" && !isBuiltinNetworkMode(svc.NetworkMode) {
			if _, ok := networks[svc.NetworkMode]; !ok {
				networks[svc.NetworkMode] = types.NetworkConfig{External: true}
			}
		}
	}
	return networks
}

// ─── 辅助函数 ───

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

// envToSlice 将 compose MappingWithEquals 转为 KEY=VALUE 列表
// value 为 nil 表示"不显式赋值"，此处跳过
func envToSlice(env types.MappingWithEquals) []string {
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
	slices.Sort(result)
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
		if k, v, ok := strings.Cut(e, "="); ok && k != "" {
			result[k] = &v
		} else {
			result[e] = nil
		}
	}
	return result
}

// buildServiceLabels 构建 compose 服务标签基础集合（复制用户标签并写入 project/service 标识）。
// extraLabels 用于追加额外的标签键值对（如 docker 专用的 container-number、oneoff）。
func buildServiceLabels(project *types.Project, svc types.ServiceConfig, extraLabels map[string]string) map[string]string {
	labels := make(map[string]string, len(svc.Labels)+2+len(extraLabels))
	maps.Copy(labels, svc.Labels)
	if project != nil && project.Name != "" {
		labels[ComposeProjectLabel] = project.Name
	}
	labels[ComposeServiceLabel] = svc.Name
	maps.Copy(labels, extraLabels)
	return labels
}
