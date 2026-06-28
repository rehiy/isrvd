// Package shell 提供 Web 终端业务服务
package shell

import (
	"context"
	"io"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/rehiy/libgo/command"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"

	"isrvd/internal/service/wsterm"
)

// Service Web 终端业务服务
type Service struct{}

// NewService 创建 Web 终端业务服务
func NewService() *Service {
	return &Service{}
}

// RunTerminal 在指定 homeDir 下启动终端并与 WebSocket 连接桥接。
// 优先使用 PTY 模式（非 Windows），失败时降级到 Pipe 模式。
func (s *Service) RunTerminal(conn *websocket.ServerConn, shell, homeDir string) {
	shell = command.GetShell(shell)
	ctx := context.Background()

	// PTY 模式（仅非 Windows）
	if runtime.GOOS != "windows" {
		cmd := command.NewCommand(ctx, shell, nil, homeDir)
		ptmx, err := pty.Start(cmd)
		if err == nil {
			defer ptmx.Close()
			handleIO(conn, ptmx, ptmx, cmd)
			return
		}
		logman.Warn("PTY 启动失败，降级到 Pipe 模式", "error", err)
		conn.Write([]byte("[提示: PTY 模式不可用，已降级到 Pipe 模式]\r\n"))
	}

	// Pipe 模式
	cmd := command.NewCommand(ctx, shell, nil, homeDir)
	if err := runWithPipe(conn, cmd); err != nil {
		logman.Error("Pipe 模式启动失败", "shell", shell, "error", err)
		conn.Write([]byte("[启动 " + shell + " 失败: " + err.Error() + "]\r\n"))
	}
}

func runWithPipe(conn *websocket.ServerConn, cmd *exec.Cmd) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return err
	}
	// handleIO 阻塞直到连接断开，返回后再关闭 stdin，避免提前关闭导致写入失败
	handleIO(conn, stdin, stdout, cmd)
	stdin.Close()
	return nil
}

func handleIO(conn *websocket.ServerConn, stdin io.Writer, stdout io.Reader, cmd *exec.Cmd) {
	// 终端 WSS 保活心跳，避免空闲被中间层断开
	stop := wsterm.KeepAlive(conn, wsterm.HeartbeatInterval)
	defer stop()

	// 确保进程被终止
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// 读取输出
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
			}
			if err != nil {
				logman.Error("shell handleIO: stdout.Read error", "error", err)
				return
			}
		}
	}()

	conn.Write([]byte("[终端已连接，输入命令后回车]\r\n"))

	// 读取输入
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logman.Error("shell handleIO: conn.Read error", "error", err)
			return
		}
		if n > 0 {
			if _, err = stdin.Write(buf[:n]); err != nil {
				logman.Error("shell handleIO: stdin.Write error", "error", err)
				return
			}
		}
	}
}
