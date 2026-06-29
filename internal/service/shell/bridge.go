package shell

import (
	"io"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/relay"
	"github.com/rehiy/libgo/websocket"
)

// BridgeOptions 定义终端 WebSocket 双向桥接的可选行为。
type BridgeOptions struct {
	Name    string
	Welcome string
	Close   func()
	Cleanup func()
}

// Bridge 在 WebSocket 与终端输入/输出流之间做双向转发。
// stdin/stdout 可来自本地 PTY、进程管道或容器 exec 会话。
func Bridge(conn *websocket.ServerConn, stdin io.Writer, stdout io.Reader, opt BridgeOptions) {
	name := opt.Name
	if name == "" {
		name = "terminal"
	}
	stop := websocket.KeepAlive(conn, 25*time.Second)
	defer stop()

	if opt.Welcome != "" {
		_, _ = conn.Write([]byte(opt.Welcome))
	}

	err := relay.Bridge(
		relay.NewEndpoint(conn, conn, conn),
		relay.NewEndpoint(stdout, stdin, terminalCloser(stdin, stdout, opt.Close)),
	)
	if err != nil && err != io.EOF {
		logman.Error("shell bridge error", "name", name, "error", err)
	}

	if opt.Cleanup != nil {
		opt.Cleanup()
	}
}

type closerFunc func()

func (fn closerFunc) Close() error {
	fn()
	return nil
}

func terminalCloser(stdin io.Writer, stdout io.Reader, closeFn func()) io.Closer {
	if closeFn != nil {
		return closerFunc(closeFn)
	}
	return multiCloser{closerFrom(stdout), closerFrom(stdin)}
}

func closerFrom(v any) io.Closer {
	closer, _ := v.(io.Closer)
	return closer
}

type multiCloser []io.Closer

func (closers multiCloser) Close() error {
	var firstErr error
	for _, closer := range closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
