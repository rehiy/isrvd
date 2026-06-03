package account

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// ─── Passkey 凭证存储（会话级）───

// passkeySession 存储 WebAuthn 会话数据
type passkeySession struct {
	Challenge     string    `yaml:"challenge" json:"challenge"`
	UserID        string    `yaml:"userId" json:"userId"`
	Username      string    `yaml:"username" json:"username"`
	DisplayName   string    `yaml:"displayName" json:"displayName"`
	ExpiresAt     time.Time `yaml:"expiresAt" json:"expiresAt"`
	CredentialIDs [][]byte  `yaml:"credentialIds" json:"credentialIds,omitempty"` // 用于登录时过滤已有凭证
}

// ─── WebAuthn 实例创建 ─────────

// getWebAuthn 创建 WebAuthn 实例
func (s *Service) getWebAuthn() (*webauthn.WebAuthn, error) {
	if config.Passkey == nil {
		logman.Warn("Passkey 未启用: config.Passkey is nil")
		return nil, fmt.Errorf("passkey 未启用")
	}
	if !config.Passkey.Enabled {
		logman.Warn("Passkey 未启用: Enabled=false", "config", fmt.Sprintf("%+v", config.Passkey))
		return nil, fmt.Errorf("passkey 未启用")
	}

	// 构建 Origin 列表
	origins := make([]string, 0, len(config.Passkey.RPOrigins))
	for _, origin := range config.Passkey.RPOrigins {
		origins = append(origins, origin)
	}

	// 设置超时时间（毫秒转纳秒）
	timeout := config.Passkey.Timeout
	if timeout == 0 {
		timeout = 60000 // 默认 60 秒
	}
	timeoutDuration := time.Duration(timeout) * time.Millisecond

	wconfig := &webauthn.Config{
		RPDisplayName: config.Passkey.RPName,
		RPID:          config.Passkey.RPID,
		RPOrigins:     origins,
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Timeout: timeoutDuration},
			Registration: webauthn.TimeoutConfig{Timeout: timeoutDuration},
		},
	}

	return webauthn.New(wconfig)
}

// ─── 用户模型适配 ────────────

// passkeyUser 实现 webauthn.User 接口
type passkeyUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte {
	return u.ID
}

func (u *passkeyUser) WebAuthnName() string {
	return u.Name
}

func (u *passkeyUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func (u *passkeyUser) WebAuthnIcon() string {
	return ""
}

// getPasskeyUser 从 MemberConfig 构建 webauthn.User
func getPasskeyUser(username string, member *config.MemberConfig) *passkeyUser {
	userID := []byte(username) // 使用用户名作为用户 ID

	credentials := make([]webauthn.Credential, 0, len(member.Passkeys))
	for _, pk := range member.Passkeys {
		credID, _ := base64.URLEncoding.DecodeString(pk.IDBase64)
		pubKey, _ := base64.URLEncoding.DecodeString(pk.PublicKeyBase64)
		aaguid, _ := base64.URLEncoding.DecodeString(pk.Authenticator.AAGUIDBase64)

		cred := webauthn.Credential{
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
		}
		credentials = append(credentials, cred)
	}

	return &passkeyUser{
		ID:          userID,
		Name:        username,
		DisplayName: member.Username,
		Credentials: credentials,
	}
}

// ─── 会话存储（内存，生产可用 Redis）───

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

// ─── 请求/响应类型 ────────────

// PasskeyBeginRegistrationRequest 开始注册请求
type PasskeyBeginRegistrationRequest struct {
	Username string `json:"username" binding:"required"`
}

// PasskeyBeginRegistrationResponse 开始注册响应
type PasskeyBeginRegistrationResponse struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"` // 使用 any 类型避免类型问题
}

// PasskeyFinishRegistrationRequest 完成注册请求
type PasskeyFinishRegistrationRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
}

// PasskeyBeginLoginRequest 开始登录请求
type PasskeyBeginLoginRequest struct {
	Username string `json:"username"` // 可选，为空则允许任何已注册用户
}

// PasskeyBeginLoginResponse 开始登录响应
type PasskeyBeginLoginResponse struct {
	SessionID string `json:"sessionId"`
	Options   any    `json:"options"` // 使用 any 类型避免类型问题
}

// PasskeyFinishLoginRequest 完成登录请求
type PasskeyFinishLoginRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
}

// ─── Service 方法 ─────────────

// PasskeyBeginRegistration 开始 Passkey 注册流程
func (s *Service) PasskeyBeginRegistration(c *gin.Context, req PasskeyBeginRegistrationRequest) (*PasskeyBeginRegistrationResponse, error) {
	member, exists := config.Members[req.Username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	user := getPasskeyUser(req.Username, member)

	// 生成注册选项 - 使用简单的 BeginRegistration
	credential, sessionData, err := w.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("开始注册失败: %w", err)
	}

	// 保存会话
	sessionID := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%d", req.Username, time.Now().UnixNano())))

	// 将 challenge 转换为 base64 存储
	challengeStr := ""
	if sessionData.Challenge != "" {
		challengeStr = base64.URLEncoding.EncodeToString([]byte(sessionData.Challenge))
	}

	savePasskeySession(sessionID, &passkeySession{
		Challenge:   challengeStr,
		UserID:      base64.URLEncoding.EncodeToString([]byte(req.Username)),
		Username:    req.Username,
		DisplayName: member.Username,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
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

	// 重建 sessionData
	challengeStr := session.Challenge
	sessionData := webauthn.SessionData{
		Challenge: challengeStr,
	}

	// 完成注册 - 使用 gin.Context 的 Request
	credential, err := w.FinishRegistration(user, sessionData, c.Request)
	if err != nil {
		return fmt.Errorf("完成注册失败: %w", err)
	}

	// 保存凭证到配置
	pk := &config.PasskeyCredential{
		IDBase64:        base64.URLEncoding.EncodeToString(credential.ID),
		PublicKeyBase64: base64.URLEncoding.EncodeToString(credential.PublicKey),
		AttestationType: credential.AttestationType,
		Authenticator: struct {
			AAGUID       []byte `yaml:"aaguid" json:"-"`
			AAGUIDBase64 string `yaml:"aaguidBase64" json:"aaguidBase64"`
			SignCount    uint32 `yaml:"signCount" json:"signCount"`
			CloneWarning bool   `yaml:"cloneWarning" json:"cloneWarning"`
		}{
			AAGUIDBase64: base64.URLEncoding.EncodeToString(credential.Authenticator.AAGUID),
			SignCount:    credential.Authenticator.SignCount,
			CloneWarning: credential.Authenticator.CloneWarning,
		},
		Flags: struct {
			UserPresent    bool `yaml:"userPresent" json:"userPresent"`
			UserVerified   bool `yaml:"userVerified" json:"userVerified"`
			BackupEligible bool `yaml:"backupEligible" json:"backupEligible"`
			BackupState    bool `yaml:"backupState" json:"backupState"`
		}{
			UserPresent:    credential.Flags.UserPresent,
			UserVerified:   credential.Flags.UserVerified,
			BackupEligible: credential.Flags.BackupEligible,
			BackupState:    credential.Flags.BackupState,
		},
		DisplayName: session.DisplayName,
		AddedAt:     time.Now(),
	}

	member.Passkeys = append(member.Passkeys, pk)
	config.Members[session.Username] = member

	// TODO: 持久化配置
	logman.Info("Passkey registered", "username", session.Username)

	return nil
}

// PasskeyBeginLogin 开始 Passkey 登录流程
func (s *Service) PasskeyBeginLogin(c *gin.Context, req PasskeyBeginLoginRequest) (*PasskeyBeginLoginResponse, error) {
	w, err := s.getWebAuthn()
	if err != nil {
		return nil, err
	}

	var user *passkeyUser
	var credentialIDs [][]byte

	if req.Username != "" {
		// 指定用户登录
		member, exists := config.Members[req.Username]
		if !exists {
			return nil, fmt.Errorf("用户不存在")
		}
		user = getPasskeyUser(req.Username, member)

		// 收集凭证 ID
		for _, cred := range user.Credentials {
			credentialIDs = append(credentialIDs, cred.ID)
		}

		// 生成断言选项
		assertion, sessionData, err := w.BeginLogin(user)
		if err != nil {
			return nil, fmt.Errorf("开始登录失败: %w", err)
		}

		// 保存会话
		sessionID := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("login-%d", time.Now().UnixNano())))

		challengeStr := ""
		if sessionData.Challenge != "" {
			challengeStr = base64.URLEncoding.EncodeToString([]byte(sessionData.Challenge))
		}

		savePasskeySession(sessionID, &passkeySession{
			Challenge:     challengeStr,
			UserID:        req.Username,
			Username:      req.Username,
			ExpiresAt:     time.Now().Add(5 * time.Minute),
			CredentialIDs: credentialIDs,
		})

		return &PasskeyBeginLoginResponse{
			SessionID: sessionID,
			Options:   assertion,
		}, nil
	} else {
		// 用户身份未知，使用可发现凭证（discoverable credentials）
		assertion, sessionData, err := w.BeginDiscoverableLogin()
		if err != nil {
			return nil, fmt.Errorf("开始登录失败: %w", err)
		}

		// 保存会话
		sessionID := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("discoverable-%d", time.Now().UnixNano())))

		challengeStr := ""
		if sessionData.Challenge != "" {
			challengeStr = base64.URLEncoding.EncodeToString([]byte(sessionData.Challenge))
		}

		savePasskeySession(sessionID, &passkeySession{
			Challenge: challengeStr,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})

		return &PasskeyBeginLoginResponse{
			SessionID: sessionID,
			Options:   assertion,
		}, nil
	}
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

	// 确定用户
	var username string
	if session.Username != "" {
		username = session.Username
	} else {
		// 从凭证 ID 查找用户
		// TODO: 实现凭证查找逻辑
		return nil, fmt.Errorf("需要指定用户名")
	}

	member, exists := config.Members[username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	user := getPasskeyUser(username, member)

	// 重建 sessionData
	sessionData := webauthn.SessionData{
		Challenge: session.Challenge,
	}

	// 完成登录验证 - 使用 gin.Context 的 Request
	_, err = w.FinishLogin(user, sessionData, c.Request)
	if err != nil {
		return nil, fmt.Errorf("验证失败: %w", err)
	}

	// 签发 JWT Token
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
	if !exists {
		return false
	}
	return len(member.Passkeys) > 0
}

// PasskeyListCredentials 查询用户的 Passkey 凭证列表
func (s *Service) PasskeyListCredentials(username string) ([]map[string]any, error) {
	member, exists := config.Members[username]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	credentials := make([]map[string]any, 0, len(member.Passkeys))
	for i, pk := range member.Passkeys {
		credentials = append(credentials, map[string]any{
			"id":           pk.IDBase64,
			"displayName":  pk.DisplayName,
			"addedAt":      pk.AddedAt,
			"signCount":    pk.Authenticator.SignCount,
			"credentialId": pk.IDBase64, // 用于删除
			"index":        i,           // 用于删除
		})
	}

	return credentials, nil
}

// PasskeyDeleteCredential 删除用户的指定 Passkey 凭证
func (s *Service) PasskeyDeleteCredential(username string, credentialID string) error {
	member, exists := config.Members[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	// 查找并删除凭证
	found := false
	newPasskeys := make([]*config.PasskeyCredential, 0, len(member.Passkeys))
	for _, pk := range member.Passkeys {
		if pk.IDBase64 == credentialID {
			found = true
			continue // 跳过这个凭证，即删除它
		}
		newPasskeys = append(newPasskeys, pk)
	}

	if !found {
		return fmt.Errorf("凭证不存在")
	}

	member.Passkeys = newPasskeys
	config.Members[username] = member

	logman.Info("Passkey credential deleted", "username", username, "credentialID", credentialID)
	return nil
}
