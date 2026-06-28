// Package wsterm 提供终端 WebSocket 连接的保活心跳。
package wsterm

import (
	"sync"
	"time"

	"github.com/rehiy/libgo/websocket"
)

// HeartbeatInterval 终端 WSS 保活间隔，与仓库内 SSE 心跳（pkgs/swarm、pkgs/docker）一致。
const HeartbeatInterval = 25 * time.Second

// KeepAlive 周期性向 conn 发送 WS ping 控制帧，保持空闲终端的 WSS 连接活跃，
// 避免被中间层（nginx/Caddy 的 proxy_read_timeout、NAT）按空闲断开。
// 浏览器自动回 pong，两端均不可见，不影响 PTY/SSH 数据流。
// 返回 stop()（多次调用安全）；ping 出错时心跳自动停止。
func KeepAlive(conn *websocket.ServerConn, interval time.Duration) (stop func()) {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if err := conn.Ping(nil); err != nil {
					return
				}
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}
