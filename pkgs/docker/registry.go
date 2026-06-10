package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/rehiy/libgo/logman"
)

// RegistryCreate 添加仓库
func (s *DockerService) RegistryCreate(reg *RegistryConfig) error {
	if reg == nil {
		return fmt.Errorf("registry config is nil")
	}
	if reg.Name == "" || reg.URL == "" {
		return fmt.Errorf("仓库名称和地址不能为空")
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if s.indexOfRegistry(reg.URL) >= 0 {
		return fmt.Errorf("仓库地址已存在: %s", reg.URL)
	}
	s.config.Registries = append(s.config.Registries, reg)
	return nil
}

// RegistryUpdate 更新仓库（根据 originalURL 定位）
func (s *DockerService) RegistryUpdate(originalURL string, reg *RegistryConfig) error {
	if reg == nil {
		return fmt.Errorf("registry config is nil")
	}
	if reg.Name == "" || reg.URL == "" {
		return fmt.Errorf("仓库名称和地址不能为空")
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	idx := s.indexOfRegistry(originalURL)
	if idx < 0 {
		return fmt.Errorf("仓库不存在: %s", originalURL)
	}
	// URL 变更时需检查新 URL 冲突
	if reg.URL != originalURL {
		if s.indexOfRegistry(reg.URL) >= 0 {
			return fmt.Errorf("仓库地址已存在: %s", reg.URL)
		}
	}
	s.config.Registries[idx] = reg
	return nil
}

// RegistryGetPassword 获取指定 URL 仓库的当前密码（供上层在更新时保留原密码使用）
func (s *DockerService) RegistryGetPassword(url string) string {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	idx := s.indexOfRegistry(url)
	if idx < 0 {
		return ""
	}
	return s.config.Registries[idx].Password
}

// RegistryDelete 删除仓库
func (s *DockerService) RegistryDelete(url string) error {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	idx := s.indexOfRegistry(url)
	if idx < 0 {
		return fmt.Errorf("仓库不存在: %s", url)
	}
	s.config.Registries = append(s.config.Registries[:idx], s.config.Registries[idx+1:]...)
	return nil
}

// ImagePush 推送镜像到仓库
func (s *DockerService) ImagePush(ctx context.Context, imageRef, registryURL, namespace string) (string, string, error) {
	// 提取镜像的短名称
	imageName := imageRef
	if idx := strings.LastIndex(imageName, "/"); idx >= 0 {
		imageName = imageName[idx+1:]
	}

	// 构建完整的目标镜像引用
	host := registryHost(registryURL)
	var targetRef string
	if namespace != "" {
		targetRef = host + "/" + namespace + "/" + imageName
	} else {
		targetRef = host + "/" + imageName
	}
	if !strings.Contains(targetRef, ":") {
		targetRef += ":latest"
	}

	// 先给镜像打标签
	if err := s.client.ImageTag(ctx, imageRef, targetRef); err != nil {
		logman.Error("Tag image for push failed", "image", imageRef, "target", targetRef, "error", err)
		return "", targetRef, err
	}

	// 推送镜像（认证信息从 targetRef 的 host 自动匹配）
	reader, err := s.client.ImagePush(ctx, targetRef, image.PushOptions{
		RegistryAuth: s.getRegistryAuth(targetRef),
	})
	if err != nil {
		logman.Error("Push image failed", "image", targetRef, "error", err)
		return "", targetRef, err
	}
	defer reader.Close()

	lastMessage, err := consumeImageStream(json.NewDecoder(reader))
	if err != nil {
		logman.Error("Push image stream error", "image", targetRef, "error", err)
		return "", targetRef, err
	}

	logman.Info("Image pushed", "image", imageRef, "target", targetRef)
	return lastMessage, targetRef, nil
}

// ImagePull 从仓库拉取镜像到本地
// RegistryURL 为空时直接从 Docker Hub / daemon 配置的 mirror 拉取
func (s *DockerService) ImagePull(ctx context.Context, imageName, registryURL, namespace string) (string, string, error) {
	// 构建完整镜像引用
	var imageRef string
	if registryURL == "" {
		// 无私有仓库：直接使用镜像名，依赖 daemon mirror 配置
		imageRef = imageName
		if !strings.Contains(imageRef, ":") && !strings.Contains(imageRef, "@") {
			imageRef += ":latest"
		}
	} else {
		// 拼接私有仓库完整引用
		host := registryHost(registryURL)
		if namespace != "" {
			imageRef = host + "/" + namespace + "/" + imageName
		} else {
			imageRef = host + "/" + imageName
		}
		if !strings.Contains(imageName, ":") && !strings.Contains(imageName, "@") {
			imageRef += ":latest"
		}
	}

	lastMsg, err := s.imagePull(ctx, imageRef)
	if err != nil {
		return "", imageRef, err
	}

	logman.Info("Image pulled from registry", "image", imageRef, "registry", registryURL)
	return lastMsg, imageRef, nil
}

// getRegistryAuth 根据镜像引用自动匹配已配置的 registry 认证信息
// imageRef 可以是完整镜像引用（如 "csighub.tencentyun.com/ns/app:v1"）或仓库 URL
// 匹配规则：提取 imageRef 的 host 部分，与已配置仓库的 host 对比
func (s *DockerService) getRegistryAuth(imageRef string) string {
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

// imagePull 执行镜像拉取，认证信息从 imageRef 的 host 自动匹配已配置的 registry
func (s *DockerService) imagePull(ctx context.Context, imageRef string) (string, error) {
	reader, err := s.client.ImagePull(ctx, imageRef, image.PullOptions{
		RegistryAuth: s.getRegistryAuth(imageRef),
	})
	if err != nil {
		logman.Error("Pull image failed", "image", imageRef, "error", err)
		return "", fmt.Errorf("拉取镜像 %s 失败: %w", imageRef, err)
	}
	defer reader.Close()

	msg, err := consumeImageStream(json.NewDecoder(reader))
	if err != nil {
		logman.Error("Pull image stream error", "image", imageRef, "error", err)
		return "", fmt.Errorf("拉取镜像 %s 失败: %w", imageRef, err)
	}
	return msg, nil
}

// indexOfRegistry 查找仓库索引（调用方负责持锁）
func (s *DockerService) indexOfRegistry(url string) int {
	for i, r := range s.config.Registries {
		if r.URL == url {
			return i
		}
	}
	return -1
}
