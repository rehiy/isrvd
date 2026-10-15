package node

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rehiy/libgo/strutil"
)

var (
	// ErrInvalidCode 注册码无效、已使用或已过期
	ErrInvalidCode = errors.New("注册码无效或已过期")
	// ErrUnauthorized 节点凭据无效
	ErrUnauthorized = errors.New("节点凭据无效")
	// ErrRateLimited 请求过于频繁
	ErrRateLimited = errors.New("请求过于频繁，请稍后再试")
)

// EnrollRequest agent 使用注册码发起注册
type EnrollRequest struct {
	Code     string `json:"code" binding:"required"` // 注册码
	Name     string `json:"name"`                    // 节点名称（可选，默认取主机名）
	Hostname string `json:"hostname"`                // 主机名
	OS       string `json:"os"`                      // 操作系统
	Arch     string `json:"arch"`                    // CPU 架构
}

// EnrollResponse 注册结果，令牌仅返回一次。
type EnrollResponse struct {
	NodeID string `json:"nodeId"` // 节点 ID
	Token  string `json:"token"`  // 节点令牌
}

// Enroll 用注册码注册新节点并签发令牌；注册码使用一次即失效。
func (s *Service) Enroll(ip string, req *EnrollRequest) (*EnrollResponse, error) {
	if !s.enrollLimit.allow(ip) {
		return nil, ErrRateLimited
	}
	s.enrollLimit.record(ip)

	submitted := strings.TrimSpace(req.Code)
	var matched *EnrollCode
	for _, code := range s.codeOpenAll(s.codes.list()) {
		if secretsEqual(code.Code, submitted) {
			matched = code
		}
	}
	if matched == nil {
		return nil, ErrInvalidCode
	}
	if time.Now().After(matched.ExpiresAt) {
		return nil, ErrInvalidCode
	}
	secret, err := randToken(32)
	if err != nil {
		return nil, err
	}
	// 先核销再创建节点；并发使用同一注册码时只有一个能核销成功
	if err := s.codes.remove(matched.ID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCode
		}
		return nil, err
	}
	id := strutil.NewString()
	hostname := clip(req.Hostname, 128)
	name := clip(matched.Name, 64)
	if name == "" {
		name = clip(req.Name, 64)
	}
	if name == "" {
		name = hostname
	}
	if name == "" {
		name = "node-" + id[:4]
	}
	n := &Node{
		ID:        id,
		Name:      name,
		Status:    StatusActive,
		Hostname:  hostname,
		OS:        clip(req.OS, 64),
		Arch:      clip(req.Arch, 64),
		CreatedAt: time.Now(),
		TokenHash: hashSecret(secret),
	}
	if err := s.nodes.insert(n); err != nil {
		return nil, err
	}
	logger.Info("节点已注册", "id", id, "name", name, "ip", ip)
	return &EnrollResponse{NodeID: id, Token: id + "." + secret}, nil
}

// ─── 限流 ───

// rateLimiter 按来源 IP 在滑动窗口内计数，纯内存实现。
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

// allow 判断窗口内的计数是否仍低于上限
func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key, time.Now())) < l.limit
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
	cutoff := now.Add(-l.window)
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

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// secretsEqual 常量时间比较两个密钥（或哈希）；任一为空视为不相等
func secretsEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// randToken 生成密码学安全的随机令牌。libgo 的 strutil.Rand 基于 math/rand，不能用于凭据。
func randToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// clip 去除首尾空白并按字符数截断
func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
