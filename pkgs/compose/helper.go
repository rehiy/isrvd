package compose

import (
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

const (
	ComposeProjectLabel         = "com.docker.compose.project"
	ComposeServiceLabel         = "com.docker.compose.service"
	ComposeContainerNumberLabel = "com.docker.compose.container-number"
	ComposeOneoffLabel          = "com.docker.compose.oneoff"
)

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

// CollectNetworks 收集 project 中所有需要确保存在的网络名（去重）
// 包含：自定义 network_mode + 各 service.networks 解析后的真实名称
func CollectNetworks(project *types.Project) []string {
	set := map[string]struct{}{}
	for _, svc := range project.Services {
		if svc.NetworkMode != "" && !isBuiltinNetworkMode(svc.NetworkMode) {
			set[svc.NetworkMode] = struct{}{}
		}
		for k := range svc.Networks {
			set[resolveNetworkName(project, k)] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
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

// defaultString 若 v 为空则返回 def
func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
