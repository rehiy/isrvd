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

// ─── 会话存储 ─────────────────

// passkeySession 存储 WebAuthn 会话数据
type passkeySession struct {
	Challenge string    `yaml:"challenge" json:"challenge"`
	Username  string    `yaml:"username" json:"username"`
	ExpiresAt time.Time `yaml:"expiresAt" json:"expiresAt"`
}

var (
	passkeySessionsMu sync.Mutex
	passkeySessions   = make(map[string]*passkeySession)
)

func init() {
	// 定期清理过期会话
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			passkeySessionsMu.Lock()
			now := time.Now()
			for k, v := range passkeySessions {
				if v.ExpiresAt.Before(now) {
					delete(passkeySessions, k)
				}
			}
			passkeySessionsMu.Unlock()
		}
	}()
}

func savePasskeySession(sessionID string, session *passkeySession) {
	passkeySessionsMu.Lock()
	defer passkeySessionsMu.Unlock()
	passkeySessions[sessionID] = session
}

func loadPasskeySession(sessionID string) *passkeySession {
	passkeySessionsMu.Lock()
	defer passkeySessionsMu.Unlock()
	return passkeySessions[sessionID]
}

func deletePasskeySession(sessionID string) {
	passkeySessionsMu.Lock()
	defer passkeySessionsMu.Unlock()
	delete(passkeySessions, sessionID)
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

// ─── 请求/响应类型 ────────────

// PasskeyBeginRegistrationRequest 开始注册请求（username 由认证中间件注入，无需请求体）
type PasskeyBeginRegistrationRequest struct{}

// PasskeyBeginRegistrationResponse 开始注册响应
type PasskeyBeginRegistrationResponse struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"`
}

// PasskeyFinishRegistrationRequest 完成注册请求
type PasskeyFinishRegistrationRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
}

// PasskeyBeginLoginRequest 开始登录请求
type PasskeyBeginLoginRequest struct {
	Username string `json:"username"` // 可选，为空则使用可发现凭证
}

// PasskeyBeginLoginResponse 开始登录响应
type PasskeyBeginLoginResponse struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"`
}

// PasskeyFinishLoginRequest 完成登录请求
type PasskeyFinishLoginRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
}

// ─── Service 方法 ─────────────

// PasskeyBeginRegistration 开始 Passkey 注册流程
func (s *Service) PasskeyBeginRegistration(c *gin.Context, req PasskeyBeginRegistrationRequest) (*PasskeyBeginRegistrationResponse, error) {
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
	savePasskeySession(sessionID, &passkeySession{
		Challenge: sessionData.Challenge,
		Username:  username,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	return &PasskeyBeginRegistrationResponse{
		SessionID: sessionID,
		Options:   credential,
	}, nil
}

// PasskeyFinishRegistration 完成 Passkey 注册
func (s *Service) PasskeyFinishRegistration(c *gin.Context, req PasskeyFinishRegistrationRequest) error {
	session := loadPasskeySession(req.SessionID)
	if session == nil {
		return fmt.Errorf("注册会话不存在或已过期")
	}
	defer deletePasskeySession(req.SessionID)

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

	credential, err := w.FinishRegistration(user, sessionData, c.Request)
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
func (s *Service) PasskeyBeginLogin(c *gin.Context, req PasskeyBeginLoginRequest) (*PasskeyBeginLoginResponse, error) {
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

		savePasskeySession(sessionID, &passkeySession{
			Challenge: sessionData.Challenge,
			Username:  req.Username,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})

		return &PasskeyBeginLoginResponse{SessionID: sessionID, Options: assertion}, nil
	}

	// 用户身份未知，使用可发现凭证
	assertion, sessionData, err := w.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("开始登录失败: %w", err)
	}

	savePasskeySession(sessionID, &passkeySession{
		Challenge: sessionData.Challenge,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	return &PasskeyBeginLoginResponse{SessionID: sessionID, Options: assertion}, nil
}

// PasskeyFinishLogin 完成 Passkey 登录
func (s *Service) PasskeyFinishLogin(c *gin.Context, req PasskeyFinishLoginRequest) (*LoginResponse, error) {
	session := loadPasskeySession(req.SessionID)
	if session == nil {
		return nil, fmt.Errorf("登录会话不存在或已过期")
	}
	defer deletePasskeySession(req.SessionID)

	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	sessionData := webauthn.SessionData{Challenge: session.Challenge}

	var username string
	if session.Username != "" {
		// 指定用户登录
		username = session.Username
		member, exists := config.Members[username]
		if !exists {
			return nil, fmt.Errorf("用户不存在")
		}
		user := getPasskeyUser(username, member)
		if _, err = w.FinishLogin(user, sessionData, c.Request); err != nil {
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
		if _, err = w.FinishDiscoverableLogin(handler, sessionData, c.Request); err != nil {
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
