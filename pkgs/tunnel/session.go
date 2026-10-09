package tunnel

import (
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

// Session 是一条已建立的隧道会话，基于 WebSocket 之上的 yamux 多路复用。
type Session struct {
	sess *yamux.Session
}

// Attach 在已升级的 WebSocket 连接上建立会话（中控侧，作为 yamux Client 发起流）。
// conn 通常是 libgo websocket.ServerConn，须已设置为二进制消息。
func Attach(conn io.ReadWriteCloser) (*Session, error) {
	return newSession(conn, true)
}

// newSession 在字节流上建立 yamux 会话；client 为 true 时作为发起流的一端。
func newSession(conn io.ReadWriteCloser, client bool) (*Session, error) {
	var (
		sess *yamux.Session
		err  error
	)
	if client {
		sess, err = yamux.Client(conn, sessionConfig())
	} else {
		sess, err = yamux.Server(conn, sessionConfig())
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Session{sess: sess}, nil
}

func sessionConfig() *yamux.Config {
	c := yamux.DefaultConfig()
	c.EnableKeepAlive = true
	c.KeepAliveInterval = 15 * time.Second
	c.ConnectionWriteTimeout = 15 * time.Second
	c.StreamOpenTimeout = 10 * time.Second
	c.StreamCloseTimeout = 30 * time.Second
	c.MaxStreamWindowSize = 1 << 20 // WebSocket 链路延迟较高，放大窗口以提升大文件吞吐
	c.LogOutput = io.Discard
	return c
}

// Open 发起一条新的流，可直接作为 http.Transport 的 DialContext 返回值。
func (s *Session) Open() (net.Conn, error) { return s.sess.Open() }

// Accept 接受对端发起的一条流。
func (s *Session) Accept() (net.Conn, error) { return s.sess.Accept() }

// Listener 返回以 Accept 接受流的 net.Listener，供 http.Server.Serve 使用。
func (s *Session) Listener() net.Listener { return listener{s} }

// Ping 测量到对端的往返延迟。
func (s *Session) Ping() (time.Duration, error) { return s.sess.Ping() }

// Done 在会话结束（对端断开、保活超时或本端关闭）时关闭。
func (s *Session) Done() <-chan struct{} { return s.sess.CloseChan() }

// Close 关闭会话及全部流。
func (s *Session) Close() error { return s.sess.Close() }

type listener struct{ s *Session }

func (l listener) Accept() (net.Conn, error) { return l.s.Accept() }
func (l listener) Close() error              { return l.s.Close() }
func (l listener) Addr() net.Addr            { return addr("tunnel") }

type addr string

func (a addr) Network() string { return "tunnel" }
func (a addr) String() string  { return string(a) }
