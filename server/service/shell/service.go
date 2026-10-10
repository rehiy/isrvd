// Package shell 提供 Web 终端业务服务
package shell

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/rehiy/libgo/command"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/i18n"
)

// Service Web 终端业务服务
type Service struct{}

// NewService 创建 Web 终端业务服务
func NewService() *Service {
	return &Service{}
}

// RunTerminal 在指定 homeDir 下启动终端并与 WebSocket 连接桥接。
// 优先使用 PTY 模式（非 Windows），失败时降级到 Pipe 模式。
func (s *Service) RunTerminal(ctx context.Context, conn *websocket.ServerConn, shell, homeDir string) {
	shell = command.GetShell(shell)

	// PTY 模式（仅非 Windows）
	if runtime.GOOS != "windows" {
		cmd := command.NewCommand(ctx, shell, nil, homeDir)
		ptmx, err := pty.Start(cmd)
		if err == nil {
			defer ptmx.Close()
			Bridge(conn, ptmx, ptmx, BridgeOptions{
				Name:    "shell",
				Welcome: i18n.TC(ctx, "[终端已连接，输入命令后回车]") + "\r\n",
				Resize: func(cols, rows int) {
					if err := pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
						logman.Warn("PTY resize failed", "cols", cols, "rows", rows, "error", err)
					}
				},
				Cleanup: func() {
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
						_ = cmd.Wait()
					}
				},
			})
			return
		}
		logman.Warn("PTY 启动失败，降级到 Pipe 模式", "error", err)
		conn.Write([]byte(i18n.TC(ctx, "[提示: PTY 模式不可用，已降级到 Pipe 模式]") + "\r\n"))
	}

	// Pipe 模式
	cmd := command.NewCommand(ctx, shell, nil, homeDir)
	if err := runWithPipe(ctx, conn, cmd); err != nil {
		logman.Error("Pipe 模式启动失败", "shell", shell, "error", err)
		conn.Write([]byte(fmt.Sprintf(i18n.TC(ctx, "[启动 %s 失败: %s]"), shell, i18n.TC(ctx, err.Error())) + "\r\n"))
	}
}

// ─── 辅助函数 ───

func runWithPipe(ctx context.Context, conn *websocket.ServerConn, cmd *exec.Cmd) error {
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
	// Bridge 阻塞直到连接断开，返回后再关闭 stdin，避免提前关闭导致写入失败
	Bridge(conn, stdin, stdout, BridgeOptions{
		Name:    "shell",
		Welcome: i18n.TC(ctx, "[终端已连接，输入命令后回车]") + "\r\n",
		Cleanup: func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		},
	})
	stdin.Close()
	return nil
}
