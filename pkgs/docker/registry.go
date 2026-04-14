package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rehiy/pango/logman"
)

// ListRegistries 列出已配置的镜像仓库
func (s *DockerService) ListRegistries() []*RegistryInfo {
	var registries []*RegistryInfo
	for _, r := range s.config.Registries {
		registries = append(registries, &RegistryInfo{
			Name:     r.Name,
			URL:      r.URL,
			Username: r.Username,
		})
	}
	return registries
}

// getRegistryAuth 获取仓库认证信息
func (s *DockerService) getRegistryAuth(registryURL string) string {
	for _, r := range s.config.Registries {
		if r.URL == registryURL {
			if r.Username != "" && r.Password != "" {
				authConfig := types.AuthConfig{
					Username:      r.Username,
					Password:      r.Password,
					ServerAddress: r.URL,
				}
				authJSON, _ := json.Marshal(authConfig)
				return base64.URLEncoding.EncodeToString(authJSON)
			}
			break
		}
	}
	return ""
}

// PushImage 推送镜像到仓库
func (s *DockerService) PushImage(ctx context.Context, req ImagePushRequest) (string, string, error) {
	// 提取镜像的短名称
	imageName := req.Image
	if idx := strings.LastIndex(imageName, "/"); idx >= 0 {
		imageName = imageName[idx+1:]
	}

	// 构建完整的目标镜像引用
	var targetRef string
	if req.Namespace != "" {
		targetRef = req.RegistryURL + "/" + req.Namespace + "/" + imageName
	} else {
		targetRef = req.RegistryURL + "/" + imageName
	}
	if !strings.Contains(targetRef, ":") {
		targetRef += ":latest"
	}

	// 先给镜像打标签
	if err := s.client.ImageTag(ctx, req.Image, targetRef); err != nil {
		logman.Error("Tag image for push failed", "image", req.Image, "target", targetRef, "error", err)
		return "", targetRef, err
	}

	// 获取认证信息
	authStr := s.getRegistryAuth(req.RegistryURL)

	// 推送镜像
	reader, err := s.client.ImagePush(ctx, targetRef, types.ImagePushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		logman.Error("Push image failed", "image", targetRef, "error", err)
		return "", targetRef, err
	}
	defer reader.Close()

	var lastMessage string
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			logman.Error("Push image stream error", "image", targetRef, "error", msg.Error)
			return "", targetRef, fmt.Errorf(msg.Error)
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pushed", "image", req.Image, "target", targetRef)
	return lastMessage, targetRef, nil
}

// PullFromRegistry 从仓库拉取镜像到本地
func (s *DockerService) PullFromRegistry(ctx context.Context, req ImagePullFromRegistryRequest) (string, string, error) {
	// 构建完整的镜像引用
	var imageRef string
	if req.Namespace != "" {
		imageRef = req.RegistryURL + "/" + req.Namespace + "/" + req.Image
	} else {
		imageRef = req.RegistryURL + "/" + req.Image
	}
	if !strings.Contains(req.Image, ":") && !strings.Contains(req.Image, "@") {
		imageRef += ":latest"
	}

	// 获取仓库认证信息
	authStr := s.getRegistryAuth(req.RegistryURL)

	// 从仓库拉取镜像到本地
	reader, err := s.client.ImagePull(ctx, imageRef, types.ImagePullOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		logman.Error("Pull image from registry failed", "image", imageRef, "error", err)
		return "", imageRef, err
	}
	defer reader.Close()

	var lastMessage string
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			logman.Error("Pull image stream error", "image", imageRef, "error", msg.Error)
			return "", imageRef, fmt.Errorf(msg.Error)
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pulled from registry", "image", imageRef, "registry", req.RegistryURL)
	return lastMessage, imageRef, nil
}
