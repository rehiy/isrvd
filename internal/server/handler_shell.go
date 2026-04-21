package server

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

	"isrvd/config"
	"isrvd/internal/helper"
)

func (app *App) shellWebSocket(c *gin.Context) {
	username := c.GetString("username")
	member, ok := config.Members[username]
	if !ok || !member.AllowTerminal {
		logman.Warn("Terminal access denied", "username", username)
		helper.RespondError(c, http.StatusForbidden, "终端访问被拒绝")
		return
	}

	shell := c.DefaultQuery("shell", "bash")
	conn, err := helper.WsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logman.Error("WebSocket 升级错误", "error", err)
		return
	}
	defer conn.Close()

	shellRunTerminal(conn, shell, member.HomeDirectory)
}

func shellRunTerminal(conn *websocket.Conn, shell, homeDir string) {
	originalShell := shell
	shell = shellResolve(shell)
	if shell != originalShell {
		shellSendMsg(conn, "[提示: "+originalShell+" 不可用，已降级到 "+shell+"]\r\n")
	}

	cmd := shellBuildCmd(shell, homeDir)

	if runtime.GOOS != "windows" {
		ptmx, err := pty.Start(cmd)
		if err == nil {
			defer ptmx.Close()
			shellHandleIO(conn, ptmx, ptmx, cmd)
			return
		}
		logman.Warn("PTY 启动失败，降级到 Pipe 模式", "error", err)
		shellSendMsg(conn, "[提示: PTY 模式不可用，已降级到 Pipe 模式]\r\n")
	}

	if err := shellRunWithPipe(conn, cmd); err != nil {
		logman.Error("Pipe 模式启动失败", "shell", shell, "error", err)
		shellSendMsg(conn, "[启动 "+shell+" 失败: "+err.Error()+"]\r\n")
	}
}

func shellRunWithPipe(conn *websocket.Conn, cmd *exec.Cmd) error {
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
	shellHandleIO(conn, stdin, stdout, cmd)
	return nil
}

func shellHandleIO(conn *websocket.Conn, stdin io.Writer, stdout io.Reader, cmd *exec.Cmd) {
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				shellSendMsg(conn, string(buf[:n]))
			}
			if err != nil {
				return
			}
		}
	}()
	shellSendMsg(conn, "[终端已连接，输入命令后回车]\r\n")
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if _, err = stdin.Write(msg); err != nil {
			return
		}
	}
}

func shellSendMsg(conn *websocket.Conn, msg string) {
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

func shellResolve(shell string) string {
	if _, err := exec.LookPath(shell); err == nil {
		return shell
	}
	switch runtime.GOOS {
	case "windows":
		for _, fb := range []string{"powershell", "pwsh", "cmd"} {
			if _, err := exec.LookPath(fb); err == nil {
				return fb
			}
		}
	case "darwin":
		for _, fb := range []string{"zsh", "bash", "sh"} {
			if _, err := exec.LookPath(fb); err == nil {
				return fb
			}
		}
	default:
		for _, fb := range []string{"bash", "sh", "zsh"} {
			if _, err := exec.LookPath(fb); err == nil {
				return fb
			}
		}
	}
	return shell
}

func shellBuildCmd(shell, homeDir string) *exec.Cmd {
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
