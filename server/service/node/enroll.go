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
)

var (
	ErrAlreadyOnline = errors.New("节点已在线，请勿重复连接")
	// ErrUnauthorized 节点凭据无效
	ErrUnauthorized = errors.New("节点凭据无效")
	// ErrRateLimited 请求过于频繁
	ErrRateLimited = errors.New("请求过于频繁，请稍后再试")
)

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
