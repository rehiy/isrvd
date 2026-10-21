package node

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrAlreadyOnline = errors.New("节点已在线，请勿重复连接")
	// ErrUnauthorized 节点凭据无效
	ErrUnauthorized = errors.New("节点凭据无效")
	// ErrRateLimited 请求过于频繁
	ErrRateLimited = errors.New("请求过于频繁，请稍后再试")
)

// Authenticate 按完整 token 的哈希查找节点，并在同一把锁内激活及预占握手。
// token 只通过 Authorization 头传递，不放在 URL 中。
func (s *Service) Authenticate(r *http.Request, ip string) (*Node, error) {
	if !s.authLimit.allow(ip) {
		return nil, ErrRateLimited
	}
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || len(token) < 64 || len(token) > 96 {
		logger.Warn("节点接入被拒绝", "reason", "missing_or_invalid_authorization", "remote", ip)
		s.authLimit.record(ip)
		return nil, ErrUnauthorized
	}
	tokenHash := hashToken(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		logger.Warn("节点接入被拒绝", "reason", "service_closed", "remote", ip)
		return nil, ErrUnauthorized
	}
	n := s.nodes.find(func(n *Node) bool { return secretsEqual(n.TokenHash, tokenHash) })
	if n == nil || (n.Status != StatusPending && n.Status != StatusActive) {
		reason := "token_not_found"
		if n != nil {
			reason = "invalid_node_status"
		}
		logger.Warn("节点接入被拒绝", "reason", reason, "remote", ip)
		s.authLimit.record(ip)
		return nil, ErrUnauthorized
	}
	if s.conns[n.ID] != nil || s.connecting[n.ID] != nil {
		return nil, ErrAlreadyOnline
	}
	if n.Status == StatusPending {
		activated, err := s.nodes.update(n.ID, func(n *Node) error {
			if n.Status != StatusPending || !secretsEqual(n.TokenHash, tokenHash) {
				return ErrUnauthorized
			}
			n.Status = StatusActive
			return nil
		})
		if err != nil {
			logger.Warn("节点激活失败", "id", n.ID, "error", err)
			return nil, ErrUnauthorized
		}
		n = activated
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

// rateLimiter 按来源 IP 在滑动窗口内计数，纯内存实现。
type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

// allow 判断窗口内的计数是否仍低于上限
func (l *rateLimiter) allow(key string) bool {
	const maxFailures = 10
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key, time.Now())) < maxFailures
}

// record 记录一次计数
func (l *rateLimiter) record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.hits) > 4096 {
		for k := range l.hits {
			l.prune(k, now)
		}
	}
	l.hits[key] = append(l.prune(key, now), now)
}

func (l *rateLimiter) prune(key string, now time.Time) []time.Time {
	times := l.hits[key]
	cutoff := now.Add(-time.Minute)
	i := 0
	for i < len(times) && times[i].Before(cutoff) {
		i++
	}
	times = times[i:]
	if len(times) == 0 {
		delete(l.hits, key)
		return nil
	}
	l.hits[key] = times
	return times
}

// ─── 辅助函数 ───

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// secretsEqual 常量时间比较两个密钥（或哈希）；任一为空视为不相等
func secretsEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
