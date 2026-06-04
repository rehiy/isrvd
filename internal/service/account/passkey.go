package account

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// PasskeyBeginRegistrationRequest 开始注册请求（username 由认证中间件注入，无需请求体）
type PasskeyBeginRegistrationRequest struct{}

// PasskeyBeginData 开始注册/登录的统一数据
type PasskeyBeginData struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"`
}

// PasskeyFinishData 完成注册/登录的统一数据
type PasskeyFinishData struct {
	SessionID  string `json:"sessionId" binding:"required"`
	Credential any    `json:"credential" binding:"required"`
}

// PasskeyBeginLoginRequest 开始登录请求
type PasskeyBeginLoginRequest struct {
	Username string `json:"username"` // 可选，为空则使用可发现凭证
}

// ─── 公开业务方法 ──────────

// PasskeyBeginRegistration 开始 Passkey 注册流程
func (s *Service) PasskeyBeginRegistration(c *gin.Context, req PasskeyBeginRegistrationRequest) (*PasskeyBeginData, error) {
	username := c.GetString("username")
	member, exists := config.Members[username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	user := getPasskeyUser(username, member)
	credential, sessionData, err := w.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("开始注册失败: %w", err)
	}

	sessionID := newSessionID()
	s.passkeyStore.save(sessionID, &passkeySession{
		Challenge: sessionData.Challenge,
		Username:  username,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	return &PasskeyBeginData{
		SessionID: sessionID,
		Options:   credential,
	}, nil
}

// PasskeyFinishRegistration 完成 Passkey 注册
func (s *Service) PasskeyFinishRegistration(c *gin.Context, req PasskeyFinishData) error {
	session := s.passkeyStore.load(req.SessionID)
	if session == nil {
		return fmt.Errorf("注册会话不存在或已过期")
	}
	defer s.passkeyStore.delete(req.SessionID)

	w, err := s.getWebAuthn()
	if err != nil {
		return err
	}

	member, exists := config.Members[session.Username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	user := getPasskeyUser(session.Username, member)
	sessionData := webauthn.SessionData{Challenge: session.Challenge}

	// 将前端序列化的凭证重建为 HTTP Request，供 webauthn 库解析
	credBody, err := json.Marshal(req.Credential)
	if err != nil {
		return fmt.Errorf("凭证序列化失败: %w", err)
	}
	fakeReq := &http.Request{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(bytes.NewReader(credBody)),
	}

	credential, err := w.FinishRegistration(user, sessionData, fakeReq)
	if err != nil {
		return fmt.Errorf("完成注册失败: %w", err)
	}
	pk := &config.PasskeyCredential{
		IDBase64:        base64.URLEncoding.EncodeToString(credential.ID),
		PublicKeyBase64: base64.URLEncoding.EncodeToString(credential.PublicKey),
		AttestationType: credential.AttestationType,
		Authenticator: config.PasskeyAuthenticator{
			AAGUIDBase64: base64.URLEncoding.EncodeToString(credential.Authenticator.AAGUID),
			SignCount:    credential.Authenticator.SignCount,
			CloneWarning: credential.Authenticator.CloneWarning,
		},
		Flags: config.PasskeyFlags{
			UserPresent:    credential.Flags.UserPresent,
			UserVerified:   credential.Flags.UserVerified,
			BackupEligible: credential.Flags.BackupEligible,
			BackupState:    credential.Flags.BackupState,
		},
		DisplayName: member.Username,
		AddedAt:     time.Now(),
	}

	member.Passkeys = append(member.Passkeys, pk)
	config.Members[session.Username] = member

	if err := config.Save(); err != nil {
		logman.Error("Passkey 注册持久化失败", "username", session.Username, "err", err)
		return fmt.Errorf("保存凭证失败: %w", err)
	}

	logman.Info("Passkey registered", "username", session.Username)
	return nil
}

// PasskeyBeginLogin 开始 Passkey 登录流程
func (s *Service) PasskeyBeginLogin(c *gin.Context, req PasskeyBeginLoginRequest) (*PasskeyBeginData, error) {
	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	sessionID := newSessionID()

	if req.Username != "" {
		// 指定用户登录
		member, exists := config.Members[req.Username]
		if !exists {
			return nil, fmt.Errorf("用户不存在")
		}
		user := getPasskeyUser(req.Username, member)

		assertion, sessionData, err := w.BeginLogin(user)
		if err != nil {
			return nil, fmt.Errorf("开始登录失败: %w", err)
		}

		s.passkeyStore.save(sessionID, &passkeySession{
			Challenge: sessionData.Challenge,
			Username:  req.Username,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})

		return &PasskeyBeginData{SessionID: sessionID, Options: assertion}, nil
	}

	// 用户身份未知，使用可发现凭证
	assertion, sessionData, err := w.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("开始登录失败: %w", err)
	}

	s.passkeyStore.save(sessionID, &passkeySession{
		Challenge: sessionData.Challenge,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	return &PasskeyBeginData{SessionID: sessionID, Options: assertion}, nil
}

// PasskeyFinishLogin 完成 Passkey 登录
func (s *Service) PasskeyFinishLogin(c *gin.Context, req PasskeyFinishData) (*LoginResponse, error) {
	session := s.passkeyStore.load(req.SessionID)
	if session == nil {
		return nil, fmt.Errorf("登录会话不存在或已过期")
	}
	defer s.passkeyStore.delete(req.SessionID)

	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	sessionData := webauthn.SessionData{Challenge: session.Challenge}

	// 将前端序列化的凭证重建为 HTTP Request，供 webauthn 库解析
	credBody, err := json.Marshal(req.Credential)
	if err != nil {
		return nil, fmt.Errorf("凭证序列化失败: %w", err)
	}
	fakeReq := &http.Request{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(bytes.NewReader(credBody)),
	}

	var username string
	if session.Username != "" {
		// 指定用户登录
		username = session.Username
		member, exists := config.Members[username]
		if !exists {
			return nil, fmt.Errorf("用户不存在")
		}
		user := getPasskeyUser(username, member)
		if _, err = w.FinishLogin(user, sessionData, fakeReq); err != nil {
			return nil, fmt.Errorf("验证失败: %w", err)
		}
	} else {
		// Discoverable login：从凭证中查找用户
		var matchedUser *passkeyUser
		handler := func(rawID, userHandle []byte) (webauthn.User, error) {
			credIDStr := base64.URLEncoding.EncodeToString(rawID)
			for uname, member := range config.Members {
				for _, pk := range member.Passkeys {
					if pk.IDBase64 == credIDStr {
						username = uname
						matchedUser = getPasskeyUser(uname, member)
						return matchedUser, nil
					}
				}
			}
			return nil, fmt.Errorf("凭证不存在")
		}
		if _, err = w.FinishDiscoverableLogin(handler, sessionData, fakeReq); err != nil {
			return nil, fmt.Errorf("验证失败: %w", err)
		}
	}

	resp, err := s.IssueLoginToken(username)
	if err != nil {
		return nil, err
	}

	logman.Info("Passkey login successful", "username", username)
	return resp, nil
}

// PasskeyEnabled 检查 Passkey 是否启用
func (s *Service) PasskeyEnabled() bool {
	return config.Passkey != nil && config.Passkey.Enabled
}

// PasskeyHasCredential 检查用户是否有 Passkey 凭证
func (s *Service) PasskeyHasCredential(username string) bool {
	member, exists := config.Members[username]
	return exists && len(member.Passkeys) > 0
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
func (s *Service) PasskeyDeleteCredential(username string, credentialID string) error {
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
		logman.Error("Passkey 删除持久化失败", "username", username, "err", err)
		return fmt.Errorf("保存配置失败: %w", err)
	}

	logman.Info("Passkey credential deleted", "username", username, "credentialID", credentialID)
	return nil
}

// ─── 辅助方法与类型定义 ──────

// passkeySession 存储 WebAuthn 会话数据
type passkeySession struct {
	Challenge string    `yaml:"challenge" json:"challenge"`
	Username  string    `yaml:"username" json:"username"`
	ExpiresAt time.Time `yaml:"expiresAt" json:"expiresAt"`
}

// passkeySessionStore 封装会话存储逻辑
type passkeySessionStore struct {
	mu       sync.Mutex
	sessions map[string]*passkeySession
	cleanup  *time.Ticker
}

// newPasskeySessionStore 创建并初始化会话存储
func newPasskeySessionStore() *passkeySessionStore {
	store := &passkeySessionStore{
		sessions: make(map[string]*passkeySession),
		cleanup:  time.NewTicker(5 * time.Minute),
	}
	go store.startCleanup()
	return store
}

// startCleanup 启动定期清理协程
func (s *passkeySessionStore) startCleanup() {
	for range s.cleanup.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.sessions {
			if v.ExpiresAt.Before(now) {
				delete(s.sessions, k)
			}
		}
		s.mu.Unlock()
	}
}

// save 保存会话
func (s *passkeySessionStore) save(sessionID string, session *passkeySession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

// load 加载会话
func (s *passkeySessionStore) load(sessionID string) *passkeySession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sessionID]
}

// delete 删除会话
func (s *passkeySessionStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// stop 停止清理协程
func (s *passkeySessionStore) stop() {
	s.cleanup.Stop()
}

// newSessionID 生成随机 sessionID
func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ─── WebAuthn 实例 ────────────

// getWebAuthn 创建 WebAuthn 实例
func (s *Service) getWebAuthn() (*webauthn.WebAuthn, error) {
	if config.Passkey == nil || !config.Passkey.Enabled {
		return nil, fmt.Errorf("passkey 未启用")
	}

	timeout := time.Duration(config.Passkey.Timeout) * time.Millisecond
	wconfig := &webauthn.Config{
		RPDisplayName: config.Passkey.RPName,
		RPID:          config.Passkey.RPID,
		RPOrigins:     config.Passkey.RPOrigins,
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Timeout: timeout},
			Registration: webauthn.TimeoutConfig{Timeout: timeout},
		},
	}
	return webauthn.New(wconfig)
}

// ─── 用户模型适配 ─────────────

// passkeyUser 实现 webauthn.User 接口
type passkeyUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.ID }
func (u *passkeyUser) WebAuthnName() string                       { return u.Name }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.DisplayName }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
func (u *passkeyUser) WebAuthnIcon() string                       { return "" }

// getPasskeyUser 从 MemberConfig 构建 webauthn.User
func getPasskeyUser(username string, member *config.MemberConfig) *passkeyUser {
	credentials := make([]webauthn.Credential, 0, len(member.Passkeys))
	for _, pk := range member.Passkeys {
		credID, _ := base64.URLEncoding.DecodeString(pk.IDBase64)
		pubKey, _ := base64.URLEncoding.DecodeString(pk.PublicKeyBase64)
		aaguid, _ := base64.URLEncoding.DecodeString(pk.Authenticator.AAGUIDBase64)
		credentials = append(credentials, webauthn.Credential{
			ID:              credID,
			PublicKey:       pubKey,
			AttestationType: pk.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:       aaguid,
				SignCount:    pk.Authenticator.SignCount,
				CloneWarning: pk.Authenticator.CloneWarning,
			},
			Flags: webauthn.CredentialFlags{
				UserPresent:    pk.Flags.UserPresent,
				UserVerified:   pk.Flags.UserVerified,
				BackupEligible: pk.Flags.BackupEligible,
				BackupState:    pk.Flags.BackupState,
			},
		})
	}
	return &passkeyUser{
		ID:          []byte(username),
		Name:        username,
		DisplayName: member.Username,
		Credentials: credentials,
	}
}
