package docker

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/docker/docker/api/types/registry"
)

// ReplaceRegistries 用已提交配置的仓库列表替换运行态副本。
func (s *DockerService) ReplaceRegistries(registries []*RegistryConfig) {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	s.config.Registries = make([]*RegistryConfig, 0, len(registries))
	for _, registry := range registries {
		if registry != nil {
			s.config.Registries = append(s.config.Registries, cloneRegistry(registry))
		}
	}
}

// RegistryAuth 根据镜像引用自动匹配已配置的 registry 认证信息
// imageRef 可以是完整镜像引用（如 "csighub.tencentyun.com/ns/app:v1"）或仓库 URL
// 匹配规则：提取 imageRef 的 host 部分，与已配置仓库的 host 对比
func (s *DockerService) RegistryAuth(imageRef string) string {
	// 提取 host：取第一个 "/" 之前的部分
	host := imageRef
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	} else {
		// 无 "/" 说明是纯镜像名（如 "nginx:latest"），属于 Docker Hub，不匹配私有仓库
		return ""
	}
	// host 不含 "." 或 ":" 时视为 Docker Hub 命名空间（如 "library/nginx"），不做匹配
	if !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return ""
	}
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	for _, r := range s.config.Registries {
		if registryHost(r.URL) == host {
			if r.Username != "" && r.Password != "" {
				authConfig := registry.AuthConfig{
					Username:      r.Username,
					Password:      r.Password,
					ServerAddress: r.URL,
				}
				authJSON, _ := json.Marshal(authConfig)
				return base64.StdEncoding.EncodeToString(authJSON)
			}
			break
		}
	}
	return ""
}

// ─── 辅助函数 ───

func cloneRegistry(registry *RegistryConfig) *RegistryConfig {
	if registry == nil {
		return nil
	}
	copy := *registry
	return &copy
}
