package docker

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"

)

// ListRegistries 列出已配置的镜像仓库
func (h *DockerHandler) ListRegistries(c *gin.Context) {
	var registries []*RegistryInfo
	for _, r := range config.Docker.Registries {
		registries = append(registries, &RegistryInfo{
			Name:     r.Name,
			URL:      r.URL,
			Username: r.Username,
		})
	}
	helper.RespondSuccess(c, "Registries listed successfully", registries)
}

// getRegistryAuth 获取仓库认证信息
func (h *DockerHandler) getRegistryAuth(registryURL string) string {
	for _, r := range config.Docker.Registries {
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
func (h *DockerHandler) PushImage(c *gin.Context) {
	var req ImagePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Push image failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()

	// 提取镜像的短名称
	// 提取镜像的短名称（去掉仓库前缀，只保留镜像名和标签）
	imageName := req.Image
	// 如果镜像名包含 /，取最后的部分作为短名称（如 csighub.com/ns/app:tag -> app:tag）
	if idx := strings.LastIndex(imageName, "/"); idx >= 0 {
		imageName = imageName[idx+1:]
	}

	// 构建完整的目标镜像引用: registry.example.com/[namespace/]image:tag
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
	if err := h.dockerClient.ImageTag(ctx, req.Image, targetRef); err != nil {
		logman.Error("Tag image for push failed", "image", req.Image, "target", targetRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "镜像打标签失败: "+err.Error())
		return
	}

	// 获取认证信息
	authStr := h.getRegistryAuth(req.RegistryURL)

	// 推送镜像
	reader, err := h.dockerClient.ImagePush(ctx, targetRef, types.ImagePushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		logman.Error("Push image failed", "image", targetRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "推送镜像失败: "+err.Error())
		return
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
			helper.RespondError(c, http.StatusInternalServerError, "推送失败: "+msg.Error)
			return
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pushed", "image", req.Image, "target", targetRef)
	helper.RespondSuccess(c, "Image pushed successfully", gin.H{
		"image":  req.Image,
		"target": targetRef,
		"status": lastMessage,
	})
}

// PullFromRegistry 从仓库拉取镜像到本地
func (h *DockerHandler) PullFromRegistry(c *gin.Context) {
	var req ImagePullFromRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Pull image from registry failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()

	// 构建完整的镜像引用
	// 构建完整的镜像引用: registry.example.com/[namespace/]image:tag
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
	authStr := h.getRegistryAuth(req.RegistryURL)

	// 从仓库拉取镜像到本地
	reader, err := h.dockerClient.ImagePull(ctx, imageRef, types.ImagePullOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		logman.Error("Pull image from registry failed", "image", imageRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+err.Error())
		return
	}
	defer reader.Close()

	// 消费拉取流
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
			helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+msg.Error)
			return
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pulled from registry", "image", imageRef, "registry", req.RegistryURL)
	helper.RespondSuccess(c, "Image pulled successfully", gin.H{
		"image":    imageRef,
		"registry": req.RegistryURL,
		"status":   lastMessage,
	})
}
