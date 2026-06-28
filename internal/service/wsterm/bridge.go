package wsterm

import (
	"io"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
)

// BridgeOptions 定义终端 WebSocket 双向桥接的可选行为。
type BridgeOptions struct {
	Name              string
	Welcome           string
	HeartbeatInterval time.Duration
	Close             func()
	Cleanup           func()
}

// Bridge 在 WebSocket 与终端输入/输出流之间做双向转发。
// stdin/stdout 可来自本地 PTY、进程管道或容器 exec 会话。
func Bridge(conn *websocket.ServerConn, stdin io.Writer, stdout io.Reader, opt BridgeOptions) {
	name := opt.Name
	if name == "" {
		name = "terminal"
	}
	interval := opt.HeartbeatInterval
	if interval <= 0 {
		interval = HeartbeatInterval
	}

	stop := KeepAlive(conn, interval)
	defer stop()

	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, stdout)
		done <- err
	}()

	if opt.Welcome != "" {
		_, _ = conn.Write([]byte(opt.Welcome))
	}

	if _, err := io.Copy(stdin, conn); err != nil && err != io.EOF {
		logman.Error("wsterm input copy error", "name", name, "error", err)
	}

	if opt.Close != nil {
		opt.Close()
	}
	if opt.Cleanup != nil {
		opt.Cleanup()
	}

	if err := <-done; err != nil && err != io.EOF {
		logman.Error("wsterm output copy error", "name", name, "error", err)
	}
}
