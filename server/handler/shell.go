package handler

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

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
	username := c.GetString("username")
	member, ok := config.Members[username]
	if !ok || !member.AllowTerminal {
		helper.RespondError(c, http.StatusForbidden, "Terminal access denied")
		return
	}

	shell := c.DefaultQuery("shell", "bash")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket 升级错误", "error", err)
		return
	}
	defer conn.Close()

	h.runTerminal(conn, shell, member.HomeDirectory)
}

// runTerminal 统一的终端运行函数
func (h *ShellHandler) runTerminal(conn *websocket.Conn, shell, homeDir string) {
	originalShell := shell
	shell = resolveShell(shell)

	// 如果shell被降级，发送提示信息
	if shell != originalShell {
		h.sendMessage(conn, "[提示: "+originalShell+" 不可用，已降级到 "+shell+"]\r\n")
	}

	cmd := buildShellCmd(shell, homeDir)

	// 尝试使用PTY（支持PTY的系统）
	if runtime.GOOS != "windows" {
		ptmx, err := pty.Start(cmd)
		if err == nil {
			defer ptmx.Close()
			h.handleTerminalIO(conn, ptmx, ptmx, cmd)
			return
		}
		logman.Warn("PTY 启动失败，降级到 Pipe 模式", "error", err)
		h.sendMessage(conn, "[提示: PTY 模式不可用，已降级到 Pipe 模式]\r\n")
	}

	// 使用pipe模式
	if err := h.runWithPipe(conn, cmd); err != nil {
		logman.Error("Pipe 模式启动失败", "shell", shell, "error", err)
		h.sendErrorMessage(conn, "启动 "+shell+" 失败", err)
	}
}

// runWithPipe 使用pipe模式运行shell
func (h *ShellHandler) runWithPipe(conn *websocket.Conn, cmd *exec.Cmd) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	h.handleTerminalIO(conn, stdin, stdout, cmd)
	return nil
}

// handleTerminalIO 处理终端输入输出
func (h *ShellHandler) handleTerminalIO(conn *websocket.Conn, stdin io.Writer, stdout io.Reader, cmd *exec.Cmd) {
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	go h.forwardOutput(conn, stdout)
	h.sendMessage(conn, "[终端已连接，输入命令后回车]\r\n")
	h.forwardInput(conn, stdin)
}

// forwardOutput 转发终端输出到WebSocket
func (h *ShellHandler) forwardOutput(conn *websocket.Conn, stdout io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			h.sendMessage(conn, string(buf[:n]))
		}
		if err != nil {
			if err != io.EOF {
				logman.Error("终端读取错误", "error", err)
			}
			return
		}
	}
}

// forwardInput 转发WebSocket输入到终端
func (h *ShellHandler) forwardInput(conn *websocket.Conn, stdin io.Writer) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logman.Error("WebSocket 读取错误", "error", err)
			return
		}
		if _, err = stdin.Write(msg); err != nil {
			logman.Error("终端写入错误", "error", err)
			return
		}
	}
}

// sendMessage 发送消息到WebSocket
func (h *ShellHandler) sendMessage(conn *websocket.Conn, msg string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		logman.Error("WebSocket 写入错误", "error", err)
	}
}

// sendErrorMessage 发送错误消息到WebSocket
func (h *ShellHandler) sendErrorMessage(conn *websocket.Conn, context string, err error) {
	h.sendMessage(conn, "["+context+": "+err.Error()+"]\r\n")
}

// resolveShell 解析可用的shell
func resolveShell(shell string) string {
	if _, err := exec.LookPath(shell); err == nil {
		return shell
	}

	// 平台特定的默认shell
	switch runtime.GOOS {
	case "windows":
		for _, fallback := range []string{"powershell", "pwsh", "cmd"} {
			if _, err := exec.LookPath(fallback); err == nil {
				return fallback
			}
		}
	case "darwin":
		for _, fallback := range []string{"zsh", "bash", "sh"} {
			if _, err := exec.LookPath(fallback); err == nil {
				return fallback
			}
		}
	default:
		for _, fallback := range []string{"bash", "sh", "zsh"} {
			if _, err := exec.LookPath(fallback); err == nil {
				return fallback
			}
		}
	}
	return shell
}

// buildShellCmd 构建shell命令
func buildShellCmd(shell, homeDir string) *exec.Cmd {
	cmd := exec.Command(shell)
	cmd.Dir = homeDir

	env := os.Environ()
	switch runtime.GOOS {
	case "windows":
		env = append(env, "TERM=dumb")
	case "darwin":
		env = append([]string{"TERM=xterm-256color", "CLICOLOR=1"}, env...)
	default:
		env = append([]string{"TERM=xterm-256color"}, env...)
	}
	cmd.Env = env

	return cmd
}
