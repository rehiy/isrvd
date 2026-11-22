package account

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	passkeySessionMaxSize    = 100
	passkeySessionMaxPerUser = 5
)

// passkeySession 存储注册/登录过程中的临时会话数据
type passkeySession struct {
	Data        webauthn.SessionData
	Username    string
	DisplayName string
	ExpiresAt   time.Time
}

// passkeySessionStore 管理注册/登录临时会话
type passkeySessionStore struct {
	mu       sync.Mutex
	sessions map[string]*passkeySession
	ticker   *time.Ticker
	done     chan struct{}
}

func newPasskeySessionStore() *passkeySessionStore {
	store := &passkeySessionStore{
		sessions: make(map[string]*passkeySession),
		done:     make(chan struct{}),
	}
	store.ticker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-store.ticker.C:
				store.mu.Lock()
				now := time.Now()
				for k, v := range store.sessions {
					if v.ExpiresAt.Before(now) {
						delete(store.sessions, k)
					}
				}
				store.mu.Unlock()
			case <-store.done:
				return
			}
		}
	}()
	return store
}

// Stop 停止会话清理协程。
func (s *passkeySessionStore) Stop() {
	s.ticker.Stop()
	close(s.done)
}

func (s *passkeySessionStore) save(id string, sess *passkeySession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	userSessions := 0
	for key, value := range s.sessions {
		if value.ExpiresAt.Before(now) {
			delete(s.sessions, key)
			continue
		}
		if sess.Username != "" && value.Username == sess.Username {
			userSessions++
		}
	}
	if len(s.sessions) >= passkeySessionMaxSize ||
		(sess.Username != "" && userSessions >= passkeySessionMaxPerUser) {
		return fmt.Errorf("Passkey 会话过多，请稍后重试")
	}
	s.sessions[id] = sess
	return nil
}

// pop 取出并删除会话（一次性消费，防重放）
func (s *passkeySessionStore) pop(id string) *passkeySession {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[id]
	if sess == nil || sess.ExpiresAt.Before(time.Now()) {
		delete(s.sessions, id)
		return nil
	}
	delete(s.sessions, id)
	return sess
}

// newSessionID 生成随机 session ID（32 位十六进制字符串）
func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
