// Package node 提供多节点管理：中控侧的节点令牌、隧道会话与请求代理，
// 以及受管机（agent）侧的常连客户端。
package node

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/strutil"

	"isrvd/pkgs/secretbox"
	"isrvd/server/config"
)

// ProtocolVersion 中控与 agent 之间的协议版本，二者必须一致才会转发请求。
const ProtocolVersion = 1

const (
	defaultCodeTTL = 60 * time.Minute
	maxCodeTTL     = 7 * 24 * time.Hour
)

var logger = logman.Named("node")

// Service 中控侧节点服务
type Service struct {
	nodes *listStore[Node]
	codes *listStore[EnrollCode]

	mu         sync.RWMutex
	conns      map[string]*nodeConn // 节点 ID → 当前在线会话
	connecting map[string]*Node     // 握手期间预占，防止并发连接
	closed     bool

	authLimit *rateLimiter // 令牌认证：按来源 IP 限制失败次数
}

// NewService 创建中控侧节点服务；节点表与注册码保存在业务数据存储中（后端见 config.OpenData）
func NewService() (*Service, error) {
	nodes, err := newListStore("node-list.yml", func(n *Node) string { return n.ID })
	if err != nil {
		return nil, fmt.Errorf("初始化节点存储失败: %w", err)
	}
	codes, err := newListStore("node-code.yml", func(c *EnrollCode) string { return c.ID })
	if err != nil {
		return nil, fmt.Errorf("初始化注册码存储失败: %w", err)
	}
	return &Service{
		nodes:      nodes,
		codes:      codes,
		conns:      map[string]*nodeConn{},
		connecting: map[string]*Node{},
		authLimit:  newRateLimiter(10, time.Minute),
	}, nil
}

// Close 关闭全部在线会话，应在应用退出或重载时调用
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	conns := s.conns
	s.conns = map[string]*nodeConn{}
	clear(s.connecting)
	s.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
}

// ─── 节点管理 ───

// NodeList 列出全部节点，附带在线状态
func (s *Service) NodeList() []*Node {
	list := s.nodes.list()
	slices.SortFunc(list, func(a, b *Node) int { return a.CreatedAt.Compare(b.CreatedAt) })
	for _, n := range list {
		s.decorate(n)
	}
	return list
}

// NodeInspect 查看指定节点，不存在返回 nil
func (s *Service) NodeInspect(id string) *Node {
	n := s.nodes.get(id)
	if n == nil {
		return nil
	}
	s.decorate(n)
	return n
}

// NodeUpdateRequest 节点更新请求
type NodeUpdateRequest struct {
	Name string `json:"name"` // 节点名称
}

// NodeUpdate 重命名节点
func (s *Service) NodeUpdate(id string, req *NodeUpdateRequest) (*Node, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return nil, fmt.Errorf("节点名称长度须为 1–64 个字符")
	}
	if err := s.nodes.update(id, func(n *Node) error {
		n.Name = name
		return nil
	}); err != nil {
		return nil, fmt.Errorf("更新节点失败: %w", err)
	}
	logger.Info("节点已重命名", "id", id, "name", name)
	return s.NodeInspect(id), nil
}

// NodeRevoke 吊销节点：令牌立即失效并断开在线会话
func (s *Service) NodeRevoke(id string) error {
	s.mu.Lock()
	err := s.nodes.update(id, func(n *Node) error {
		n.Status = StatusRevoked
		n.TokenHash = ""
		return nil
	})
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("吊销节点失败: %w", err)
	}
	c := s.disconnectLocked(id)
	s.mu.Unlock()
	if c != nil {
		c.close()
	}
	logger.Info("节点已吊销", "id", id)
	return nil
}

// NodeDelete 删除节点并断开在线会话
func (s *Service) NodeDelete(id string) error {
	s.mu.Lock()
	if err := s.codes.remove(id); err != nil && !errors.Is(err, ErrNotFound) {
		s.mu.Unlock()
		return fmt.Errorf("删除节点失败: %w", err)
	}
	err := s.nodes.remove(id)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("删除节点失败: %w", err)
	}
	c := s.disconnectLocked(id)
	s.mu.Unlock()
	if c != nil {
		c.close()
	}
	logger.Info("节点已删除", "id", id)
	return nil
}

// ─── 注册码管理 ───

// EnrollCodeCreateRequest 注册码创建请求
type EnrollCodeCreateRequest struct {
	Name       string `json:"name"`       // 备注
	TTLMinutes int    `json:"ttlMinutes"` // 有效期（分钟），默认 60，最长 10080
}

// EnrollCodeList 列出尚未使用且未过期的注册码（含可用于接入的明文）
func (s *Service) EnrollCodeList() []*EnrollCode {
	aead, err := s.codeAEAD()
	if err != nil {
		logger.Warn("清理注册码失败", "error", err)
		return nil
	}
	now := time.Now()
	list := []*EnrollCode{}
	if err := s.codes.removeIf(func(c *EnrollCode) bool {
		if now.After(c.ExpiresAt) || s.nodes.get(c.ID) != nil {
			return true
		}
		plain, err := secretbox.Open(aead, c.Sealed)
		if err != nil {
			logger.Warn("注册码无法解密（JWT 密钥可能已变更），已清除", "id", c.ID)
			return true
		}
		opened := *c
		opened.Code = plain
		list = append(list, &opened)
		return false
	}); err != nil {
		logger.Warn("清理注册码失败", "error", err)
	}
	slices.SortFunc(list, func(a, b *EnrollCode) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return list
}

// EnrollCodeCreate 生成注册码
func (s *Service) EnrollCodeCreate(creator string, req *EnrollCodeCreateRequest) (*EnrollCode, error) {
	ttl := time.Duration(req.TTLMinutes) * time.Minute
	switch {
	case ttl <= 0:
		ttl = defaultCodeTTL
	case ttl > maxCodeTTL:
		ttl = maxCodeTTL
	}
	id := strutil.NewString()
	secret, err := randToken(32)
	if err != nil {
		return nil, fmt.Errorf("生成注册码失败: %w", err)
	}
	// 注册码即节点令牌（<节点ID>.<密钥>）：受管机首次连接时激活为节点，之后继续沿用
	code := id + "." + secret
	aead, err := s.codeAEAD()
	if err != nil {
		return nil, fmt.Errorf("加密注册码失败: %w", err)
	}
	sealed, err := secretbox.Seal(aead, code)
	if err != nil {
		return nil, fmt.Errorf("加密注册码失败: %w", err)
	}
	now := time.Now()
	item := &EnrollCode{
		ID:        id,
		Name:      clip(req.Name, 64),
		Sealed:    sealed,
		CreatedBy: creator,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	if err := s.codes.insert(item); err != nil {
		return nil, fmt.Errorf("保存注册码失败: %w", err)
	}
	logger.Info("注册码已创建", "id", id, "by", creator, "expiresAt", item.ExpiresAt)
	created := *item
	created.Code = code
	return &created, nil
}

// EnrollCodeDelete 撤销注册码
func (s *Service) EnrollCodeDelete(id string) error {
	if err := s.codes.remove(id); err != nil {
		return fmt.Errorf("删除注册码失败: %w", err)
	}
	logger.Info("注册码已撤销", "id", id)
	return nil
}

// ─── 辅助函数 ───

// codeAEAD 每次按当前 JWT 密钥派生，配置重载后无需重建服务
func (s *Service) codeAEAD() (cipher.AEAD, error) {
	return secretbox.NewAEAD("isrvd-node-code", config.Current().Server.JWTSecret)
}

// decorate 把在线状态合并到节点副本
func (s *Service) decorate(n *Node) {
	s.mu.RLock()
	c := s.conns[n.ID]
	s.mu.RUnlock()
	if c == nil {
		return
	}
	n.Online = true
	n.Latency = c.latency.Load()
	n.Compatible = c.compatible
	if c.info != nil {
		n.AgentVersion = c.info.Version
	}
	connectedAt := c.connectedAt
	n.ConnectedAt = &connectedAt
	n.RemoteAddr = c.remote
}

// disconnectLocked 在 s.mu 持锁期间摘除节点连接；调用方在解锁后关闭会话。
func (s *Service) disconnectLocked(id string) *nodeConn {
	c := s.conns[id]
	delete(s.conns, id)
	return c
}
