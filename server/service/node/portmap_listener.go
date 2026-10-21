package node

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	portmapMaxConns = 512              // 每条 TCP 规则的并发连接上限
	udpMaxSessions  = 256              // 每条 UDP 规则的并发来源上限
	udpIdleTimeout  = 60 * time.Second // UDP 来源空闲回收时间
)

// portListener 中控上一条正在监听的转发规则。目标可在规则更新时原地替换，监听不中断。
type portListener struct {
	nodeID string
	proto  string
	port   int
	target atomic.Value // string
	closer interface{ Close() error }

	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
}

func (l *portListener) targetAddr() string { return l.target.Load().(string) }

// track 登记一条客户端连接；监听已关闭或并发达到上限时拒绝。
func (l *portListener) track(c net.Conn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || len(l.conns) >= portmapMaxConns {
		return false
	}
	l.conns[c] = struct{}{}
	return true
}

func (l *portListener) untrack(c net.Conn) {
	l.mu.Lock()
	delete(l.conns, c)
	l.mu.Unlock()
}

// close 停止监听并断开已建立的连接。
func (l *portListener) close() {
	l.mu.Lock()
	l.closed = true
	conns := l.conns
	l.conns = nil
	l.mu.Unlock()
	_ = l.closer.Close()
	for c := range conns {
		_ = c.Close()
	}
}

// NodePortMapUpdate 整体替换节点的端口转发规则。先确认新增端口可监听，再持久化，最后撤销被移除的监听；
// 任一步失败都回滚，不会出现「已保存但未生效」。
func (s *Service) NodePortMapUpdate(id string, req *PortMapRequest) (*NodeInfo, error) {
	rules, err := normalizePortMaps(req.PortMaps, s.portmap)
	if err != nil {
		return nil, err
	}

	s.listenMu.Lock()
	defer s.listenMu.Unlock()

	n := s.nodes.get(id)
	if n == nil {
		return nil, ErrNotFound
	}
	if n.Status == StatusRevoked {
		return nil, errors.New("已吊销的节点不能配置端口转发")
	}
	if len(rules) > 0 && !s.agentSupportsPortMap(id) {
		return nil, errors.New("该节点程序不支持端口转发，请先升级节点")
	}
	for _, other := range s.nodes.list() {
		if other.ID == id {
			continue
		}
		for _, f := range other.PortMaps {
			if slices.ContainsFunc(rules, func(r PortMap) bool { return r.Proto == f.Proto && r.Listen == f.Listen }) {
				return nil, fmt.Errorf("监听端口 %d/%s 已被节点 %s 使用", f.Listen, f.Proto, other.Name)
			}
		}
	}

	var started []string
	rollback := func() {
		for _, key := range started {
			s.portmapStop(key)
		}
	}
	keep := map[string]bool{}
	for _, r := range rules {
		key := portmapKey(r.Proto, r.Listen)
		keep[key] = true
		if l := s.listeners[key]; l != nil {
			if l.nodeID != id {
				rollback()
				return nil, fmt.Errorf("监听端口 %d/%s 已被占用", r.Listen, r.Proto)
			}
			continue
		}
		if err := s.portmapStart(id, r); err != nil {
			rollback()
			return nil, err
		}
		started = append(started, key)
	}

	updated, err := s.nodes.update(id, func(m *Node) error {
		if m.Status == StatusRevoked { // 与 NodeRevoke 并发时以吊销为准，避免写回规则
			return errors.New("已吊销的节点不能配置端口转发")
		}
		m.PortMaps = rules
		return nil
	})
	if err != nil {
		rollback()
		return nil, fmt.Errorf("保存端口转发失败: %w", err)
	}

	for key, l := range s.listeners {
		if l.nodeID == id && !keep[key] {
			s.portmapStop(key)
		}
	}
	for _, r := range rules {
		s.listeners[portmapKey(r.Proto, r.Listen)].target.Store(r.Target)
	}
	logger.Info("端口转发已更新", "id", id, "count", len(rules))
	return s.nodeInfo(updated), nil
}

// agentSupportsPortMap 判断节点是否明确不支持端口转发：离线或未能协商出能力清单时无法判断，放行；
// 协商成功但清单里没有该能力，才视为不支持。
func (s *Service) agentSupportsPortMap(id string) bool {
	s.mu.RLock()
	c := s.conns[id]
	s.mu.RUnlock()
	return c == nil || c.info == nil || slices.Contains(c.info.Features, FeaturePortMap)
}

// portmapRestore 启动时按持久化的规则恢复监听；单条失败只告警，不影响其他规则。
func (s *Service) portmapRestore() {
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	for _, n := range s.nodes.list() {
		if n.Status == StatusRevoked {
			continue
		}
		for _, f := range n.PortMaps {
			if !s.portmap.PortAllowed(f.Listen) {
				logger.Warn("端口转发未启用或端口超出允许范围，已跳过", "node", n.ID, "listen", f.Listen, "proto", f.Proto)
				continue
			}
			if err := s.portmapStart(n.ID, f); err != nil {
				logger.Warn("恢复端口转发失败", "node", n.ID, "listen", f.Listen, "proto", f.Proto, "error", err)
			}
		}
	}
}

// portmapStopNode 撤销某节点的全部监听（吊销、删除时调用）。
func (s *Service) portmapStopNode(id string) {
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	for key, l := range s.listeners {
		if l.nodeID == id {
			s.portmapStop(key)
		}
	}
}

// portmapStopAll 撤销全部监听（服务关闭时调用）。
func (s *Service) portmapStopAll() {
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	for key := range s.listeners {
		s.portmapStop(key)
	}
}

// portmapStart 开始监听一条规则，调用方须持有 listenMu。
func (s *Service) portmapStart(nodeID string, f PortMap) error {
	addr := net.JoinHostPort(s.portmap.BindAddr(), strconv.Itoa(f.Listen))
	l := &portListener{nodeID: nodeID, proto: f.Proto, port: f.Listen, conns: map[net.Conn]struct{}{}}
	l.target.Store(f.Target)
	switch f.Proto {
	case ProtoTCP:
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("监听 %s/tcp 失败: %w", addr, err)
		}
		l.closer = ln
		go s.acceptTCP(ln, l)
	case ProtoUDP:
		pc, err := net.ListenPacket("udp", addr)
		if err != nil {
			return fmt.Errorf("监听 %s/udp 失败: %w", addr, err)
		}
		l.closer = pc
		go s.serveUDP(pc, l)
	default:
		return errors.New("转发协议只能是 tcp 或 udp")
	}
	s.listeners[portmapKey(f.Proto, f.Listen)] = l
	logger.Info("端口转发已监听", "node", nodeID, "listen", addr, "proto", f.Proto, "target", f.Target)
	return nil
}

// portmapStop 停止监听并断开其连接，调用方须持有 listenMu。
func (s *Service) portmapStop(key string) {
	if l := s.listeners[key]; l != nil {
		delete(s.listeners, key)
		l.close()
		logger.Info("端口转发已停止", "node", l.nodeID, "listen", l.port, "proto", l.proto)
	}
}

// portmapOpen 在节点的隧道上开一条转发流并发送头部；节点离线或隧道不可用时返回错误。
func (s *Service) portmapOpen(l *portListener) (net.Conn, error) {
	c, err := s.connFor(l.nodeID)
	if err != nil {
		return nil, err
	}
	stream, err := c.sess.Open()
	if err != nil {
		return nil, err
	}
	if err := writePortMapHeader(stream, l.proto, l.targetAddr()); err != nil {
		_ = stream.Close()
		return nil, err
	}
	return stream, nil
}

// ─── TCP ───

func (s *Service) acceptTCP(ln net.Listener, l *portListener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			time.Sleep(100 * time.Millisecond) // 句柄耗尽等临时错误
			continue
		}
		if !l.track(c) {
			_ = c.Close()
			continue
		}
		go func() {
			defer l.untrack(c)
			defer c.Close()
			stream, err := s.portmapOpen(l)
			if err != nil {
				return
			}
			pipe(c, stream)
		}()
	}
}

// ─── UDP ───

// udpSession 一个来源地址对应一条隧道流。
type udpSession struct {
	stream net.Conn
	out    chan []byte
	done   chan struct{}
	once   sync.Once
	last   atomic.Int64 // 最近活动时间（UnixNano）
}

func (u *udpSession) close() {
	u.once.Do(func() {
		close(u.done)
		_ = u.stream.Close()
	})
}

func (s *Service) serveUDP(pc net.PacketConn, l *portListener) {
	var mu sync.Mutex
	sessions := map[string]*udpSession{}
	stop := make(chan struct{})
	defer func() {
		close(stop)
		mu.Lock()
		for _, u := range sessions {
			u.close()
		}
		mu.Unlock()
	}()

	go func() {
		ticker := time.NewTicker(udpIdleTimeout / 2)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-udpIdleTimeout).UnixNano()
				mu.Lock()
				for key, u := range sessions {
					if u.last.Load() < cutoff {
						u.close()
						delete(sessions, key)
					}
				}
				mu.Unlock()
			}
		}
	}()

	buf := make([]byte, 65535)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		key := addr.String()
		mu.Lock()
		u := sessions[key]
		count := len(sessions)
		mu.Unlock()

		if u == nil {
			if count >= udpMaxSessions {
				continue
			}
			stream, err := s.portmapOpen(l)
			if err != nil {
				continue
			}
			u = &udpSession{stream: stream, out: make(chan []byte, 128), done: make(chan struct{})}
			mu.Lock()
			sessions[key] = u
			mu.Unlock()
			go udpPump(pc, addr, u, func() {
				mu.Lock()
				if sessions[key] == u {
					delete(sessions, key)
				}
				mu.Unlock()
			})
		}
		u.last.Store(time.Now().UnixNano())
		select {
		case u.out <- bytes.Clone(buf[:n]):
		default: // 发送队列已满：UDP 允许丢包，不阻塞其他来源
		}
	}
}

// udpPump 为一个来源地址搬运数据：out 队列 → 隧道流，隧道流 → 回写给来源；任一方向结束即回收会话。
func udpPump(pc net.PacketConn, addr net.Addr, u *udpSession, release func()) {
	defer release()
	defer u.close()

	go func() {
		defer u.close()
		for {
			select {
			case p := <-u.out:
				if writeFrame(u.stream, p) != nil {
					return
				}
			case <-u.done:
				return
			}
		}
	}()

	buf := make([]byte, 65535)
	for {
		n, err := readFrame(u.stream, buf)
		if err != nil {
			return
		}
		u.last.Store(time.Now().UnixNano())
		if _, err := pc.WriteTo(buf[:n], addr); err != nil && errors.Is(err, net.ErrClosed) {
			return
		}
	}
}
