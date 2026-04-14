package docker

import (
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// ListContainers 列出容器
func (h *DockerHandler) ListContainers(c *gin.Context) {
	ctx := c.Request.Context()
	all := c.DefaultQuery("all", "false") == "true"

	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: all})
	if err != nil {
		logman.Error("List containers failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取容器列表失败")
		return
	}

	var result []*model.ContainerInfo
	for _, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, &model.ContainerInfo{
			ID:      ct.ID[:12],
			Name:    name,
			Image:   ct.Image,
			State:   ct.State,
			Status:  ct.Status,
			Ports:   formatPorts(ct.Ports),
			Created: ct.Created,
			Labels:  ct.Labels,
		})
	}

	helper.RespondSuccess(c, "Containers listed successfully", result)
}

// ContainerAction 容器操作
func (h *DockerHandler) ContainerAction(c *gin.Context) {
	var req model.ContainerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Container action failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	ctx := c.Request.Context()

	switch req.Action {
	case "start":
		err := h.dockerClient.ContainerStart(ctx, req.ID, types.ContainerStartOptions{})
		if err != nil {
			logman.Error("Start container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "启动容器失败: "+err.Error())
			return
		}
	case "stop":
		timeout := 10
		err := h.dockerClient.ContainerStop(ctx, req.ID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			logman.Error("Stop container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "停止容器失败: "+err.Error())
			return
		}
	case "restart":
		timeout := 10
		err := h.dockerClient.ContainerRestart(ctx, req.ID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			logman.Error("Restart container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "重启容器失败: "+err.Error())
			return
		}
	case "remove":
		err := h.dockerClient.ContainerRemove(ctx, req.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logman.Error("Remove container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "删除容器失败: "+err.Error())
			return
		}
	case "pause":
		err := h.dockerClient.ContainerPause(ctx, req.ID)
		if err != nil {
			logman.Error("Pause container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "暂停容器失败: "+err.Error())
			return
		}
	case "unpause":
		err := h.dockerClient.ContainerUnpause(ctx, req.ID)
		if err != nil {
			logman.Error("Unpause container failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "恢复容器失败: "+err.Error())
			return
		}
	default:
		logman.Error("Unsupported container action", "action", req.Action, "id", req.ID)
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}

	logman.Info("Container action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, "Container "+req.Action+" successfully", nil)
}

// ContainerLogs 获取容器日志
func (h *DockerHandler) ContainerLogs(c *gin.Context) {
	var req model.ContainerLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Container logs failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	ctx := c.Request.Context()

	tailStr := req.Tail
	if tailStr == "" {
		tailStr = "100"
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true,
		Tail: tailStr, Follow: false, Timestamps: true,
	}

	reader, err := h.dockerClient.ContainerLogs(ctx, req.ID, options)
	if err != nil {
		logman.Error("Get container logs failed", "id", req.ID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取日志失败: "+err.Error())
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logman.Error("Read container logs failed", "id", req.ID, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "读取日志失败")
		return
	}

	helper.RespondSuccess(c, "Container logs retrieved", gin.H{"id": req.ID, "logs": helper.ParseDockerLogs(data)})
}
