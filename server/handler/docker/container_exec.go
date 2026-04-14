package docker

import (
	"context"
	"io"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
)

// ContainerExec 容器终端 WebSocket 处理
func (h *DockerHandler) ContainerExec(c *gin.Context) {
	containerID := c.Query("id")
	if containerID == "" {
		logman.Error("Container exec failed", "error", "container ID is required")
		helper.RespondError(c, http.StatusBadRequest, "容器 ID 不能为空")
		return
	}

	shell := c.DefaultQuery("shell", "/bin/sh")

	conn, err := helper.WsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket upgrade error", "error", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	// 创建 exec 实例
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := h.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		h.sendWsMessage(conn, "[创建终端会话失败: "+err.Error()+"]\r\n")
		return
	}

	// 连接到 exec 实例
	attachConfig := types.ExecStartCheck{Tty: true}
	hijackedResp, err := h.dockerClient.ContainerExecAttach(ctx, execResp.ID, attachConfig)
	if err != nil {
		h.sendWsMessage(conn, "[连接终端失败: "+err.Error()+"]\r\n")
		return
	}
	defer hijackedResp.Close()

	h.sendWsMessage(conn, "[容器终端已连接]\r\n")

	// 转发容器输出到 WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					logman.Error("Container exec read error", "error", err)
				}
				return
			}
			if n > 0 {
				h.sendWsMessage(conn, string(buf[:n]))
			}
		}
	}()

	// 转发 WebSocket 输入到容器
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logman.Error("WebSocket read error", "error", err)
			return
		}
		if _, err := hijackedResp.Conn.Write(msg); err != nil {
			logman.Error("Container exec write error", "error", err)
			return
		}
	}
}
