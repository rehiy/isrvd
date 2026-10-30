package node

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"slices"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/rehiy/libgo/wstunnel"
)

// nodeConn 一个在线节点的会话及其到该节点的 HTTP 代理
type nodeConn struct {
	nodeID    string
	sess      *wstunnel.Session
	transport *http.Transport
	proxy     *httputil.ReverseProxy
	latency   atomic.Int64 // 毫秒
	info      *AgentInfo   // 协商得到，attach 之前写入，之后只读；旧版受管机无协商接口时为 nil
}

func newNodeConn(nodeID string, sess *wstunnel.Session) *nodeConn {
	c := &nodeConn{nodeID: nodeID, sess: sess}
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

// negotiate 请求 agent 自身信息以获知可选能力（AgentInfo.Features）。
// 失败不影响转发：旧版受管机没有协商接口时 info 保持 nil，由节点自己对不支持的请求作答。
func (c *nodeConn) negotiate(ctx context.Context) {
	resp, err := c.get(ctx, AgentInfoPath)
	if err != nil {
		logger.Warn("节点协商失败", "id", c.nodeID, "error", err)
		return
	}
	defer resp.Body.Close()

	var envelope struct {
		Success bool      `json:"success"`
		Payload AgentInfo `json:"payload"`
	}
	if resp.StatusCode != http.StatusOK ||
		json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&envelope) != nil || !envelope.Success {
		logger.Warn("节点不支持协商，可能版本过旧", "id", c.nodeID, "status", resp.StatusCode)
		return
	}
	c.info = &envelope.Payload
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

// Serve 在已升级的 WebSocket 连接上建立隧道会话并阻塞到会话结束。调用方须先通过 Authenticate，
// 且由 libgo 的 websocket.ServerConfig 完成升级。ctx 取消（如服务重载）时主动断开。
// 同一节点只允许一个在线或正在握手的连接。
func (s *Service) Serve(ctx context.Context, ws io.ReadWriteCloser, remote string, n *Node) {
	sess, err := wstunnel.Attach(ws)
	if err != nil {
		logger.Warn("节点隧道建立失败", "id", n.ID, "error", err)
		return
	}
	conn := newNodeConn(n.ID, sess)
	conn.negotiate(ctx)
	s.syncInfo(n, conn.info)

	if !s.attach(conn, n.TokenHash) {
		conn.close()
		return
	}
	logger.Info("节点已上线", "id", n.ID, "name", n.Name, "remote", remote)
	go conn.pingLoop()

	select {
	case <-sess.Done():
	case <-ctx.Done():
	}
	s.detach(conn)
	logger.Info("节点已离线", "id", n.ID, "name", n.Name)
}

// syncInfo 把节点上报的主机名与平台信息写入记录，名称始终由用户维护。
func (s *Service) syncInfo(n *Node, info *AgentInfo) {
	if info == nil {
		return
	}
	hostname, osName, arch := clip(info.Hostname, 128), clip(info.OS, 64), clip(info.Arch, 64)
	if n.Hostname == hostname && n.OS == osName && n.Arch == arch {
		return
	}
	if _, err := s.nodes.update(n.ID, func(m *Node) error {
		m.Hostname, m.OS, m.Arch = hostname, osName, arch
		return nil
	}); err != nil {
		logger.Warn("更新节点信息失败", "id", n.ID, "error", err)
	}
}

// attach 在吊销/删除使用的同一把锁内复核令牌与状态，再登记在线会话。
func (s *Service) attach(c *nodeConn, tokenHash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.nodes.byID(c.nodeID)
	if s.closed || n == nil || n.Status != StatusActive || n.TokenHash != tokenHash || s.conns[c.nodeID] != nil {
		return false
	}
	s.conns[c.nodeID] = c
	delete(s.connecting, c.nodeID)
	return true
}

// detach 移除并关闭会话；只摘除仍属于自己的在线记录
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
		return c, nil
	}
	if s.nodes.byID(id) == nil {
		return nil, ErrNotFound
	}
	return nil, errOffline
}

// nodeInfo 将持久化快照与当前连接合并为响应，在线字段不进入存储。
func (s *Service) nodeInfo(n *Node) *NodeInfo {
	info := &NodeInfo{Node: *n}
	s.mu.RLock()
	c := s.conns[n.ID]
	s.mu.RUnlock()
	if c == nil {
		return info
	}
	info.Online, info.Latency = true, c.latency.Load()
	if c.info != nil {
		info.AgentVersion = c.info.Version
		info.Features = slices.Clone(c.info.Features)
	}
	info.Negotiated = c.info != nil
	return info
}

// ─── 辅助函数 ───

// clip 去除首尾空白并按字符数截断
func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
