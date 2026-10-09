// Package tunnel 在单条 WebSocket 连接上提供多路复用的双向流通道。
//
// 角色与谁拨号无关：agent 主动拨号（Dial）并作为 yamux Server 接受流，
// 中控在已升级的连接上 Attach 并作为 yamux Client 发起流。
// 每个流都是一条 net.Conn，上层可以直接在其上跑 http.Server / http.Transport。
//
// 中控侧的 WebSocket 升级由 libgo 的 websocket.ServerConfig 完成，本包不再重复实现。
package tunnel

import (
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsConn 把拨号得到的 WebSocket 连接适配为字节流：只收发二进制消息，读取时跨消息拼接。
//
// 这是 agent 侧唯一保留的适配。libgo 的 websocket.NewClient 无法携带 Authorization 头、
// 不支持 context 取消，也不返回握手响应码（无法区分「令牌被吊销」与网络故障），
// 且没有导出从底层连接构造 websocket.Conn 的方法，因此这里直接使用 gorilla。
type wsConn struct {
	ws     *websocket.Conn
	reader io.Reader  // 当前消息的剩余读取器；仅 Read 所在的单个 goroutine 访问
	wmu    sync.Mutex // gorilla 不允许并发写
}

func newWSConn(ws *websocket.Conn) *wsConn { return &wsConn{ws: ws} }

func (c *wsConn) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if c.reader == nil {
			messageType, r, err := c.ws.NextReader()
			if err != nil {
				return 0, err
			}
			if messageType != websocket.BinaryMessage {
				continue // 忽略非二进制消息
			}
			c.reader = r
		}
		n, err := c.reader.Read(p)
		if err == io.EOF {
			c.reader = nil
			if n == 0 {
				continue
			}
			return n, nil
		}
		return n, err
	}
}

func (c *wsConn) Write(p []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if err := c.ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsConn) Close() error {
	// 尽力发送关闭帧，失败不影响底层连接关闭
	c.wmu.Lock()
	_ = c.ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	c.wmu.Unlock()
	return c.ws.Close()
}
