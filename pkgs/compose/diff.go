package compose

import (
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/moby/docker-image-spec/specs-go/v1"
)

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
