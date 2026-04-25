package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"github.com/rehiy/pango/logman"
)

// ContainerExec 容器终端 WebSocket 处理（业务逻辑层）
func (s *DockerService) ContainerExec(conn *websocket.Conn, containerID, shell string) {
	if shell == "" {
		shell = "/bin/sh"
	}

	ctx := context.Background()

	// 创建 exec 实例
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		sendWsMessage(conn, "[创建终端会话失败: "+err.Error()+"]\r\n")
		return
	}

	// 连接到 exec 实例
	attachConfig := container.ExecStartOptions{Tty: true}
	hijackedResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, attachConfig)
	if err != nil {
		sendWsMessage(conn, "[连接终端失败: "+err.Error()+"]\r\n")
		return
	}
	defer hijackedResp.Close()

	sendWsMessage(conn, "[容器终端已连接]\r\n")

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
				sendWsMessage(conn, string(buf[:n]))
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

// sendWsMessage 发送 WebSocket 消息
func sendWsMessage(conn *websocket.Conn, msg string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		logman.Error("WebSocket write error", "error", err)
	}
}
