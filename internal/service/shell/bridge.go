package shell

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/relay"
	"github.com/rehiy/libgo/websocket"
)

const ResizeControlPrefix = "\x00isrvd:resize:"

// BridgeOptions 定义终端 WebSocket 双向桥接的可选行为。
type BridgeOptions struct {
	Name    string
	Welcome string
	Resize  func(cols, rows int)
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

	wsReader := io.Reader(conn)
	if opt.Resize != nil {
		wsReader = terminalResizeReader{reader: conn, resize: opt.Resize}
	}

	err := relay.Bridge(
		relay.NewEndpoint(wsReader, conn, conn),
		relay.NewEndpoint(stdout, stdin, terminalCloser(stdin, stdout, opt.Close)),
	)
	if err != nil && err != io.EOF {
		logman.Error("shell bridge error", "name", name, "error", err)
	}

	if opt.Cleanup != nil {
		opt.Cleanup()
	}
}

type terminalResizeReader struct {
	reader io.Reader
	resize func(cols, rows int)
}

func (r terminalResizeReader) Read(p []byte) (int, error) {
	for {
		n, err := r.reader.Read(p)
		if n > 0 {
			if cols, rows, ok := parseTerminalResize(p[:n]); ok {
				r.resize(cols, rows)
				continue
			}
		}
		return n, err
	}
}

func parseTerminalResize(data []byte) (int, int, bool) {
	msg := string(data)
	if !strings.HasPrefix(msg, ResizeControlPrefix) {
		return 0, 0, false
	}
	size := strings.TrimPrefix(msg, ResizeControlPrefix)
	parts := strings.Split(size, ":")
	if len(parts) != 2 {
		return 0, 0, true
	}
	cols, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, true
	}
	rows, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, true
	}
	if cols < 10 || rows < 3 || cols > 1000 || rows > 1000 {
		return 0, 0, true
	}
	return cols, rows, true
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
