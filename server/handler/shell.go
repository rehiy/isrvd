package handler

import (
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Shell处理器
type ShellHandler struct{}

// 创建Shell处理器
func NewShellHandler() *ShellHandler {
	return &ShellHandler{}
}

// Shell WebSocket处理
func (h *ShellHandler) WebSocket(c *gin.Context) {
	// 检查用户是否允许使用终端
	username := c.GetString("username")
	member, ok := config.Members[username]
	if !ok || !member.AllowTerminal {
		helper.RespondError(c, http.StatusForbidden, "Terminal access denied")
		return
	}

	shell := c.DefaultQuery("shell", "bash")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket upgrade error", "error", err)
		return
	}
	defer conn.Close()

	cmd, ptmx, err := h.startShell(shell, member.HomeDirectory)
	if err != nil {
		logman.Error("Start shell error", "shell", shell, "error", err)
		h.sendMessage(conn, "[启动 "+shell+" 失败]\r\n")
		return
	}
	defer func() {
		cmd.Process.Kill()
		ptmx.Close()
	}()

	go h.forwardShellOutput(conn, ptmx)
	h.sendMessage(conn, "[终端已连接，输入命令后回车]\r\n")
	h.handleUserInput(conn, ptmx)
}

// sendMessage 发送消息到WebSocket
func (h *ShellHandler) sendMessage(conn *websocket.Conn, msg string) {
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// startShell 启动shell进程
func (h *ShellHandler) startShell(shell, homeDir string) (*exec.Cmd, *os.File, error) {
	cmd := exec.Command(shell)
	cmd.Dir = homeDir
	cmd.Env = append([]string{"TERM=xterm-256color"}, os.Environ()...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, ptmx, nil
}

// forwardShellOutput 将shell输出转发到WebSocket
func (h *ShellHandler) forwardShellOutput(conn *websocket.Conn, ptmx *os.File) {
	buf := make([]byte, 1024)
	for {
		n, err := ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				logman.Error("PTY read error", "error", err)
			}
			return
		}
		if n > 0 {
			h.sendMessage(conn, string(buf[:n]))
		}
	}
}

// handleUserInput 处理用户输入并写入shell
func (h *ShellHandler) handleUserInput(conn *websocket.Conn, ptmx *os.File) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logman.Error("WebSocket read error", "error", err)
			return
		}
		if _, err = ptmx.Write(msg); err != nil {
			logman.Error("PTY write error", "error", err)
			return
		}
	}
}
