package account

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// initPasskey 初始化 WebAuthn 单例并构建内存索引，由 NewService 调用
func (s *Service) initPasskey() {
	if config.Passkey != nil && config.Passkey.Enabled {
		timeout := time.Duration(config.Passkey.Timeout) * time.Millisecond
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		w, err := webauthn.New(&webauthn.Config{
			RPDisplayName: config.Passkey.RPName,
			RPID:          config.Passkey.RPID,
			RPOrigins:     config.Passkey.RPOrigins,
			Timeouts: webauthn.TimeoutsConfig{
				Login:        webauthn.TimeoutConfig{Timeout: timeout},
				Registration: webauthn.TimeoutConfig{Timeout: timeout},
			},
		})
		if err != nil {
			logman.Error("WebAuthn 初始化失败", "err", err)
		} else {
			s.webAuthn = w
			// 从配置构建内存索引
			for username, member := range config.Members {
				for _, pk := range member.Passkeys {
					s.credIndex[pk.IDBase64] = username
					s.signCounts[pk.IDBase64] = pk.SignCount
				}
			}
		}
	}
}

// PasskeyEnabled 返回当前是否已启用 Passkey 功能
func (s *Service) PasskeyEnabled() bool {
	return s.webAuthn != nil
}

// PasskeyBeginData 开始注册/登录的统一响应
type PasskeyBeginData struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"`
}

// PasskeyBeginRegistration 开始 Passkey 注册
func (s *Service) PasskeyBeginRegistration(c *gin.Context, displayName string) (*PasskeyBeginData, error) {
	username := c.GetString("username")
	member, exists := config.Members[username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	options, sessionData, err := s.webAuthn.BeginRegistration(s.buildPasskeyUser(username, member))
	if err != nil {
		return nil, fmt.Errorf("开始注册失败: %w", err)
	}

	sessionID := newSessionID()
	s.passkeyStore.save(sessionID, &passkeySession{
		Data:        *sessionData,
		Username:    username,
		DisplayName: displayName,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	return &PasskeyBeginData{SessionID: sessionID, Options: options}, nil
}

// PasskeyFinishRegistration 完成 Passkey 注册，直接从 c.Request 读取凭证数据
func (s *Service) PasskeyFinishRegistration(c *gin.Context, sessionID string) error {
	session := s.passkeyStore.pop(sessionID)
	if session == nil {
		return fmt.Errorf("注册会话不存在或已过期")
	}

	member, exists := config.Members[session.Username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	credential, err := s.webAuthn.FinishRegistration(s.buildPasskeyUser(session.Username, member), session.Data, c.Request)
	if err != nil {
		return fmt.Errorf("完成注册失败: %w", err)
	}

	credIDStr := base64.RawURLEncoding.EncodeToString(credential.ID)

	// 防止重复注册
	for _, pk := range member.Passkeys {
		if pk.IDBase64 == credIDStr {
			return fmt.Errorf("该凭证已注册")
		}
	}

	displayName := session.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("Passkey #%s", credIDStr[:8])
	}

	member.Passkeys = append(member.Passkeys, &config.PasskeyCredential{
		IDBase64:        credIDStr,
		PublicKeyBase64: base64.RawURLEncoding.EncodeToString(credential.PublicKey),
		AAGUIDBase64:    base64.RawURLEncoding.EncodeToString(credential.Authenticator.AAGUID),
		SignCount:       credential.Authenticator.SignCount,
		DisplayName:     displayName,
		AddedAt:         time.Now(),
	})
	config.Members[session.Username] = member

	if err := config.Save(); err != nil {
		return fmt.Errorf("保存凭证失败: %w", err)
	}

	// 更新内存索引
	s.indexMu.Lock()
	s.credIndex[credIDStr] = session.Username
	s.signCounts[credIDStr] = credential.Authenticator.SignCount
	s.indexMu.Unlock()

	logman.Info("Passkey registered", "username", session.Username, "credID", safePrefix(credIDStr))
	return nil
}

// PasskeyBeginLogin 开始 Passkey 登录，username 为空则使用可发现凭证
func (s *Service) PasskeyBeginLogin(username string) (*PasskeyBeginData, error) {
	sessionID := newSessionID()

	if username != "" {
		member, exists := config.Members[username]
		if !exists {
			return nil, fmt.Errorf("用户验证失败")
		}
		options, sessionData, err := s.webAuthn.BeginLogin(s.buildPasskeyUser(username, member))
		if err != nil {
			return nil, fmt.Errorf("开始登录失败: %w", err)
		}
		s.passkeyStore.save(sessionID, &passkeySession{
			Data: *sessionData, Username: username,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})
		return &PasskeyBeginData{SessionID: sessionID, Options: options}, nil
	}

	// 可发现凭证登录
	options, sessionData, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("开始登录失败: %w", err)
	}
	s.passkeyStore.save(sessionID, &passkeySession{
		Data:      *sessionData,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
	return &PasskeyBeginData{SessionID: sessionID, Options: options}, nil
}

// PasskeyFinishLogin 完成 Passkey 登录，直接从 c.Request 读取凭证数据
func (s *Service) PasskeyFinishLogin(c *gin.Context, sessionID string) (*LoginResponse, error) {
	session := s.passkeyStore.pop(sessionID)
	if session == nil {
		return nil, fmt.Errorf("登录会话不存在或已过期")
	}

	var (
		username string
		credID   []byte
	)

	if session.Username != "" {
		username = session.Username
		member, exists := config.Members[username]
		if !exists {
			return nil, fmt.Errorf("用户不存在")
		}
		credential, err := s.webAuthn.FinishLogin(s.buildPasskeyUser(username, member), session.Data, c.Request)
		if err != nil {
			return nil, fmt.Errorf("验证失败: %w", err)
		}
		credID = credential.ID

		// 更新内存 signCount（取最大值，并做克隆检测）
		credIDStr := base64.RawURLEncoding.EncodeToString(credID)
		newCount := credential.Authenticator.SignCount

		s.indexMu.Lock()
		stored := s.signCounts[credIDStr]
		if newCount > stored {
			s.signCounts[credIDStr] = newCount
		} else if newCount != 0 && newCount <= stored {
			// 克隆检测：signCount 未增长或倒退，凭证可能被克隆
			s.indexMu.Unlock()
			return nil, fmt.Errorf("凭证可能被克隆，登录被拒绝")
		}
		s.indexMu.Unlock()
	} else {
		// Discoverable login：通过 credIndex 查找用户
		credential, err := s.webAuthn.FinishDiscoverableLogin(func(rawID, _ []byte) (webauthn.User, error) {
			s.indexMu.RLock()
			uname, ok := s.credIndex[base64.RawURLEncoding.EncodeToString(rawID)]
			s.indexMu.RUnlock()
			if !ok {
				return nil, fmt.Errorf("验证失败")
			}
			username = uname
			member, exists := config.Members[uname]
			if !exists {
				return nil, fmt.Errorf("验证失败")
			}
			return s.buildPasskeyUser(uname, member), nil
		}, session.Data, c.Request)
		if err != nil {
			return nil, fmt.Errorf("验证失败: %w", err)
		}
		credID = credential.ID

		// 更新内存 signCount（取最大值，并做克隆检测）
		credIDStr := base64.RawURLEncoding.EncodeToString(credID)
		newCount := credential.Authenticator.SignCount

		s.indexMu.Lock()
		stored := s.signCounts[credIDStr]
		if newCount > stored {
			s.signCounts[credIDStr] = newCount
		} else if newCount != 0 && newCount <= stored {
			// 克隆检测：signCount 未增长或倒退，凭证可能被克隆
			s.indexMu.Unlock()
			return nil, fmt.Errorf("凭证可能被克隆，登录被拒绝")
		}
		s.indexMu.Unlock()
	}

	resp, err := s.IssueLoginToken(username)
	if err != nil {
		return nil, err
	}
	logman.Info("Passkey login", "username", username)
	return resp, nil
}

// PasskeyListCredentials 查询用户的 Passkey 凭证列表
func (s *Service) PasskeyListCredentials(username string) ([]*config.PasskeyCredential, error) {
	member, exists := config.Members[username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}
	return member.Passkeys, nil
}

// PasskeyDeleteCredential 删除用户的指定 Passkey 凭证
func (s *Service) PasskeyDeleteCredential(username, credentialID string) error {
	member, exists := config.Members[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	newPasskeys := make([]*config.PasskeyCredential, 0, len(member.Passkeys))
	for _, pk := range member.Passkeys {
		if pk.IDBase64 != credentialID {
			newPasskeys = append(newPasskeys, pk)
		}
	}
	if len(newPasskeys) == len(member.Passkeys) {
		return ErrPasskeyNotFound
	}

	member.Passkeys = newPasskeys
	config.Members[username] = member
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	s.indexMu.Lock()
	delete(s.credIndex, credentialID)
	delete(s.signCounts, credentialID)
	s.indexMu.Unlock()
	logman.Info("Passkey deleted", "username", username, "credID", safePrefix(credentialID))
	return nil
}

// PasskeyUpdateCredentialName 更新凭证显示名称
func (s *Service) PasskeyUpdateCredentialName(username, credentialID, displayName string) error {
	member, exists := config.Members[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}
	for _, pk := range member.Passkeys {
		if pk.IDBase64 == credentialID {
			pk.DisplayName = displayName
			config.Members[username] = member
			return config.Save()
		}
	}
	return ErrPasskeyNotFound
}

// passkeyUser 实现 webauthn.User 接口
type passkeyUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.id }
func (u *passkeyUser) WebAuthnName() string                       { return u.name }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }
func (u *passkeyUser) WebAuthnIcon() string                       { return "" }

// buildPasskeyUser 构建 webauthn.User 实例，从配置和内存索引读取凭证信息
func (s *Service) buildPasskeyUser(username string, member *config.MemberConfig) *passkeyUser {
	credentials := make([]webauthn.Credential, 0, len(member.Passkeys))
	for _, pk := range member.Passkeys {
		credID, _ := base64.RawURLEncoding.DecodeString(pk.IDBase64)
		pubKey, _ := base64.RawURLEncoding.DecodeString(pk.PublicKeyBase64)
		aaguid, _ := base64.RawURLEncoding.DecodeString(pk.AAGUIDBase64)

		s.indexMu.RLock()
		count, ok := s.signCounts[pk.IDBase64]
		s.indexMu.RUnlock()
		signCount := pk.SignCount
		if ok {
			signCount = count
		}

		credentials = append(credentials, webauthn.Credential{
			ID:        credID,
			PublicKey: pubKey,
			Authenticator: webauthn.Authenticator{
				AAGUID:    aaguid,
				SignCount: signCount,
			},
		})
	}
	return &passkeyUser{
		id:          []byte(username),
		name:        username,
		displayName: member.Username,
		credentials: credentials,
	}
}

// passkeySession 存储注册/登录过程中的临时会话数据
type passkeySession struct {
	Data        webauthn.SessionData
	Username    string
	DisplayName string
	ExpiresAt   time.Time
}

const passkeySessionMaxSize = 100

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

// Stop 停止会话清理协程，防止资源泄漏
func (s *passkeySessionStore) Stop() {
	s.ticker.Stop()
	close(s.done)
}

func (s *passkeySessionStore) save(id string, sess *passkeySession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sessions) < passkeySessionMaxSize {
		s.sessions[id] = sess
	}
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

// safePrefix 返回字符串前 8 个字符，长度不足时返回全部
func safePrefix(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}
