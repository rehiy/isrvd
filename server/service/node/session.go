package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rehiy/libgo/wstunnel"

	"isrvd/pkgs/secretbox"
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
// 令牌首次出现时对应一个待接入的注册码：校验通过后就地激活为节点，之后继续用同一令牌连接。
func (s *Service) Authenticate(r *http.Request, ip string) (*Node, error) {
	if !s.authLimit.allow(ip) {
		return nil, ErrRateLimited
	}
	raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	id, secret, ok := strings.Cut(raw, ".")

	tokenHash := hashSecret(secret)
	var n *Node
	if ok {
		n = s.nodes.get(id)
		if n == nil {
			n = s.activate(id, raw, tokenHash)
		}
	}
	if n == nil || n.Status != StatusActive || !secretsEqual(n.TokenHash, tokenHash) {
		s.authLimit.record(ip)
		return nil, ErrUnauthorized
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrUnauthorized
	}
	if s.conns[n.ID] != nil || s.connecting[n.ID] != nil {
		return nil, ErrAlreadyOnline
	}
	s.connecting[n.ID] = n
	return n, nil
}

// ReleaseConnection 释放握手预占；调用方在认证成功后 defer，包含升级失败和会话退出。
func (s *Service) ReleaseConnection(n *Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connecting[n.ID] == n {
		delete(s.connecting, n.ID)
	}
}

// activate 用待接入的注册码（即完整令牌 raw）创建节点并核销该码；码不存在、已过期或不匹配返回 nil。
func (s *Service) activate(id, raw, tokenHash string) *Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := s.nodes.get(id); n != nil { // 并发连接中已有一个完成了激活
		return n
	}
	c := s.codes.get(id)
	if c == nil || time.Now().After(c.ExpiresAt) {
		return nil
	}
	aead, err := s.codeAEAD()
	if err != nil {
		return nil
	}
	plain, err := secretbox.Open(aead, c.Sealed)
	if err != nil || !secretsEqual(plain, raw) {
		return nil
	}
	n := &Node{
		ID:        id,
		Name:      clip(c.Name, 64),
		Status:    StatusActive,
		CreatedAt: time.Now(),
		TokenHash: tokenHash,
	}
	if err := s.nodes.insert(n); err != nil {
		logger.Warn("节点激活失败", "id", id, "error", err)
		return nil
	}
	if err := s.codes.remove(id); err != nil && !errors.Is(err, ErrNotFound) {
		logger.Warn("清理已接入令牌失败", "id", id, "error", err)
	}
	logger.Info("节点已接入", "id", id, "name", n.Name)
	return n
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
	conn := newNodeConn(n.ID, sess, remote)
	conn.negotiate()
	s.syncInfo(n, conn.info)

	if !s.attach(conn, n.TokenHash) {
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

// syncInfo 把节点上报的主机名与平台信息写入节点记录；名称为空时取主机名。
func (s *Service) syncInfo(n *Node, info *AgentInfo) {
	hostname, osName, arch := n.Hostname, n.OS, n.Arch
	if info != nil {
		hostname, osName, arch = clip(info.Hostname, 128), clip(info.OS, 64), clip(info.Arch, 64)
	}
	if n.Name != "" && n.Hostname == hostname && n.OS == osName && n.Arch == arch {
		return
	}
	if err := s.nodes.update(n.ID, func(m *Node) error {
		m.Hostname, m.OS, m.Arch = hostname, osName, arch
		if m.Name == "" {
			if m.Name = hostname; m.Name == "" {
				m.Name = "node-" + m.ID[:4]
			}
		}
		n.Name = m.Name
		return nil
	}); err != nil {
		logger.Warn("更新节点信息失败", "id", n.ID, "error", err)
	}
}

// attach 在吊销/删除使用的同一把锁内复核令牌与状态，再登记在线会话。
func (s *Service) attach(c *nodeConn, tokenHash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.nodes.get(c.nodeID)
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
