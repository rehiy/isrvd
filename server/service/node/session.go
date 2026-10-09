package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rehiy/libgo/wstunnel"
)

// AgentInfo agent 自身的版本与平台信息，供中控做协议协商
type AgentInfo struct {
	Mode     string `json:"mode"`     // 运行模式，agent 恒为 agent
	Version  string `json:"version"`  // 程序版本
	Protocol int    `json:"protocol"` // 协议版本，须与中控一致
	Hostname string `json:"hostname"` // 主机名
	OS       string `json:"os"`       // 操作系统
	Arch     string `json:"arch"`     // CPU 架构
}

// nodeConn 一个在线节点的会话及其到该节点的 HTTP 代理
type nodeConn struct {
	nodeID      string
	sess        *wstunnel.Session
	transport   *http.Transport
	proxy       *httputil.ReverseProxy
	remote      string
	connectedAt time.Time
	latency     atomic.Int64 // 毫秒
	info        *AgentInfo   // 协商得到，attach 之前写入，之后只读
	compatible  bool         // 协议版本一致才允许转发
}

func newNodeConn(nodeID string, sess *wstunnel.Session, remote string) *nodeConn {
	c := &nodeConn{
		nodeID:      nodeID,
		sess:        sess,
		remote:      remote,
		connectedAt: time.Now(),
	}
	// 每条 HTTP 连接对应隧道上的一条流；地址被忽略，流量只会去往这个节点
	c.transport = &http.Transport{
		DialContext:         func(context.Context, string, string) (net.Conn, error) { return sess.Open() },
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     60 * time.Second,
		DisableCompression:  true, // 不改写 Accept-Encoding，由浏览器与 agent 直接协商
	}
	c.proxy = newProxy(c.transport)
	return c
}

func (c *nodeConn) close() {
	_ = c.sess.Close()
	c.transport.CloseIdleConnections()
}

// get 经隧道对节点发起一次 GET 请求（10 秒超时），调用方负责关闭响应体
func (c *nodeConn) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://agent"+path, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: c.transport, Timeout: 10 * time.Second}
	return client.Do(req)
}

// negotiate 请求 agent 自身信息并判定协议是否兼容；失败时保持不兼容，但仍保持连接以便界面提示
func (c *nodeConn) negotiate() {
	resp, err := c.get(context.Background(), AgentInfoPath)
	if err != nil {
		logger.Warn("节点协议协商失败", "id", c.nodeID, "error", err)
		return
	}
	defer resp.Body.Close()

	var envelope struct {
		Success bool      `json:"success"`
		Payload AgentInfo `json:"payload"`
	}
	if resp.StatusCode != http.StatusOK ||
		json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&envelope) != nil || !envelope.Success {
		logger.Warn("节点不支持协议协商，可能版本过旧", "id", c.nodeID, "status", resp.StatusCode)
		return
	}
	c.info = &envelope.Payload
	c.compatible = envelope.Payload.Protocol == ProtocolVersion
	if !c.compatible {
		logger.Warn("节点协议版本不兼容", "id", c.nodeID, "agent", envelope.Payload.Protocol, "center", ProtocolVersion)
	}
}

// pingLoop 周期采样隧道延迟，会话结束时退出
func (c *nodeConn) pingLoop() {
	sample := func() {
		if rtt, err := c.sess.Ping(); err == nil {
			c.latency.Store(max(rtt.Milliseconds(), 1))
		}
	}
	sample()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.sess.Done():
			return
		case <-ticker.C:
			sample()
		}
	}
}

// Authenticate 校验 agent 握手时携带的节点令牌（Authorization: Bearer <id>.<secret>）。
// 令牌不放在 URL 中，避免进入访问日志；浏览器无法设置该头，天然无法被跨站页面利用。
func (s *Service) Authenticate(r *http.Request, ip string) (*Node, error) {
	if !s.authLimit.allow(ip) {
		return nil, ErrRateLimited
	}
	raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	id, secret, ok := strings.Cut(raw, ".")

	var n *Node
	if ok {
		n = s.nodes.get(id)
	}
	tokenHash := ""
	if n != nil {
		tokenHash = n.TokenHash
	}
	if n == nil || n.Status != StatusApproved || !secretsEqual(tokenHash, hashSecret(secret)) {
		s.authLimit.record(ip)
		return nil, ErrUnauthorized
	}
	// 首次成功连接后领取密钥作废
	if n.ClaimHash != "" {
		if err := s.nodes.update(n.ID, func(x *Node) error {
			x.ClaimHash = ""
			return nil
		}); err != nil {
			logger.Warn("清除领取密钥失败", "id", n.ID, "error", err)
		}
	}
	return n, nil
}

// Serve 在已升级的 WebSocket 连接上建立隧道会话并阻塞到会话结束。调用方须先通过 Authenticate，
// 且由 libgo 的 websocket.ServerConfig 完成升级。ctx 取消（如服务重载）时主动断开。
// 同一节点的新连接会顶替旧连接。
func (s *Service) Serve(ctx context.Context, ws io.ReadWriteCloser, remote string, n *Node) {
	sess, err := wstunnel.Attach(ws)
	if err != nil {
		logger.Warn("节点隧道建立失败", "id", n.ID, "error", err)
		return
	}
	conn := newNodeConn(n.ID, sess, remote)
	conn.negotiate()

	if !s.attach(conn) {
		conn.close()
		return
	}
	logger.Info("节点已上线", "id", n.ID, "name", n.Name, "remote", conn.remote, "compatible", conn.compatible)
	go conn.pingLoop()

	select {
	case <-sess.Done():
	case <-ctx.Done():
	}
	s.detach(conn)
	logger.Info("节点已离线", "id", n.ID, "name", n.Name)
}

// attach 登记在线会话，返回 false 表示服务已关闭
func (s *Service) attach(c *nodeConn) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	old := s.conns[c.nodeID]
	s.conns[c.nodeID] = c
	s.mu.Unlock()
	if old != nil {
		old.close()
	}
	return true
}

// detach 移除并关闭会话；若已被新连接顶替则只关闭自己
func (s *Service) detach(c *nodeConn) {
	s.mu.Lock()
	if s.conns[c.nodeID] == c {
		delete(s.conns, c.nodeID)
	}
	s.mu.Unlock()
	c.close()
}

// connFor 返回在线会话，并说明不可用的原因
func (s *Service) connFor(id string) (*nodeConn, error) {
	s.mu.RLock()
	c := s.conns[id]
	s.mu.RUnlock()
	if c != nil {
		if !c.compatible {
			peer := "未知"
			if c.info != nil {
				peer = fmt.Sprint(c.info.Protocol)
			}
			return nil, fmt.Errorf("节点版本不兼容（中控协议 %d，节点协议 %s），请升级节点程序", ProtocolVersion, peer)
		}
		return c, nil
	}
	if s.nodes.get(id) == nil {
		return nil, ErrNotFound
	}
	return nil, errOffline
}
