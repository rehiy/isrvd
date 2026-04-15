package docker

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/pkgs/docker"
	"isrvd/server/helper"
)

// DockerHandler Docker 处理器
type DockerHandler struct {
	service *docker.DockerService
}

// NewDockerHandler 创建 Docker 处理器
func NewDockerHandler() (*DockerHandler, error) {
	var registries []*docker.RegistryConfig
	for _, r := range config.Docker.Registries {
		registries = append(registries, &docker.RegistryConfig{
			Name:        r.Name,
			Description: r.Description,
			URL:         r.URL,
			Username:    r.Username,
			Password:    r.Password,
		})
	}

	cfg := &docker.DockerConfig{
		Host:          config.Docker.Host,
		ContainerRoot: config.Docker.ContainerRoot,
		Registries:    registries,
	}

	svc, err := docker.NewDockerService(cfg)
	if err != nil {
		logman.Error("Docker client init failed", "error", err)
		return nil, err
	}

	return &DockerHandler{service: svc}, nil
}

// GetClient 获取 Docker 服务
func (h *DockerHandler) GetClient() *docker.DockerService {
	return h.service
}

// CheckAvailability 检测 Docker 可用性
func (h *DockerHandler) CheckAvailability(ctx context.Context) bool {
	if h.service == nil {
		return false
	}
	_, err := h.service.GetInfo(ctx)
	return err == nil
}

// Info 获取 Docker 概览信息
func (h *DockerHandler) Info(c *gin.Context) {
	info, err := h.service.GetInfo(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "连接 Docker 失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "Docker info retrieved", info)
}

// sendWsMessage 发送消息到 WebSocket
func (h *DockerHandler) sendWsMessage(conn *websocket.Conn, msg string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		logman.Error("WebSocket write error", "error", err)
	}
}
