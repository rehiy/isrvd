package helper

import (
	"crypto/md5"
	"encoding/hex"
	"sync"
	"time"
)

var session *Session

// 会话管理器
type Session struct {
	sessions map[string]time.Time
	mutex    sync.RWMutex
}

// 创建新的会话管理器
func NewSession() *Session {
	if session != nil {
		return session
	}

	session = &Session{
		sessions: make(map[string]time.Time),
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			session.CleanupExpired()
		}
	}()

	return session
}

// 创建会话令牌
func (s *Session) CreateToken(username string) string {
	token := md5sum(username + time.Now().String())
	s.mutex.Lock()
	s.sessions[token] = time.Now().Add(24 * time.Hour)
	s.mutex.Unlock()
	return token
}

// 验证令牌
func (s *Session) ValidateToken(token string) bool {
	s.mutex.RLock()
	expiry, exists := s.sessions[token]
	s.mutex.RUnlock()

	if !exists {
		return false
	}

	if expiry.Before(time.Now()) {
		s.DeleteToken(token)
		return false
	}

	return true
}

// 删除令牌
func (s *Session) DeleteToken(token string) {
	s.mutex.Lock()
	delete(s.sessions, token)
	s.mutex.Unlock()
}

// 清理过期的会话
func (s *Session) CleanupExpired() {
	s.mutex.Lock()
	now := time.Now()
	for token, expiry := range s.sessions {
		if expiry.Before(now) {
			delete(s.sessions, token)
		}
	}
	s.mutex.Unlock()
}

// 计算MD5哈希
func md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
