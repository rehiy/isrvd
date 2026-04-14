package docker

import (
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"

)

// DockerHandler Docker处理器
type DockerHandler struct {
	dockerClient *client.Client
}

// NewDockerHandler 创建Docker处理器
func NewDockerHandler() (*DockerHandler, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if config.Docker.Host != "" {
		opts = append(opts, client.WithHost(config.Docker.Host))
	} else {
		opts = append(opts, client.FromEnv)
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &DockerHandler{dockerClient: cli}, nil
}

// GetClient 获取Docker客户端
func (h *DockerHandler) GetClient() *client.Client {
	return h.dockerClient
}

// Info 获取Docker概览信息
func (h *DockerHandler) Info(c *gin.Context) {
	ctx := c.Request.Context()

	info, err := h.dockerClient.Info(ctx)
	if err != nil {
		logman.Error("Docker info failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "连接 Docker 失败: "+err.Error())
		return
	}
	_ = info

	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		logman.Error("Container list failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败")
		return
	}

	var running, stopped, paused int64
	for _, ct := range containers {
		switch ct.State {
		case "running":
			running++
		case "paused":
			paused++
		default:
			stopped++
		}
	}

	images, _ := h.dockerClient.ImageList(ctx, types.ImageListOptions{All: true})
	volList, _ := h.dockerClient.VolumeList(ctx, volume.ListOptions{})
	networks, _ := h.dockerClient.NetworkList(ctx, types.NetworkListOptions{})

	helper.RespondSuccess(c, "Docker info retrieved", DockerInfo{
		ContainersRunning: running,
		ContainersStopped: stopped,
		ContainersPaused:  paused,
		ImagesTotal:       int64(len(images)),
		VolumesTotal:      int64(len(volList.Volumes)),
		NetworksTotal:     int64(len(networks)),
	})
}

// sendWsMessage 发送消息到 WebSocket
func (h *DockerHandler) sendWsMessage(conn *websocket.Conn, msg string) {
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
}
