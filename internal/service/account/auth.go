package account

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"

	"isrvd/config"
)

// ─── 认证入口 ──────────────

// Auth 根据配置选择认证方式，返回用户名和错误原因。
// 供中间件统一调用，避免在 server 层判断认证模式。
func (s *Service) Auth(c *gin.Context) (username, errMsg string) {
	if config.THA != nil && config.THA.Enabled {
		return s.HeaderTokenCheck(c)
	}
	return s.JWTCheck(c)
}

// AuthMix 可选认证：成功返回用户名，失败返回空字符串（不中断请求）。
func (s *Service) AuthMix(c *gin.Context) string {
	username, _ := s.Auth(c)
	return username
}

// AuthInfoResponse 认证模式及当前用户信息
type AuthInfoResponse struct {
	Mode                  string      `json:"mode"`               // 认证模式：jwt | header
	Username              string      `json:"username,omitempty"` // 当前登录用户名（未登录时为空）
	Member                *MemberInfo `json:"member,omitempty"`   // 成员详细信息
	OIDCEnabled           bool        `json:"oidcEnabled"`        // 是否启用 OIDC 登录
	OIDCBtnLabel          string      `json:"oidcBtnLabel"`       // OIDC 登录按钮文案
	PasskeyEnabled        bool        `json:"passkeyEnabled"`     // 是否启用 Passkey 登录
	PasswordDisabled  bool        `json:"passwordDisabled"`  // 是否禁用密码登录
	PasswordMinLength int         `json:"passwordMinLength"` // 密码最小长度
}

// AuthInfo 返回当前认证模式及已登录用户信息
func (s *Service) AuthInfo(username string) *AuthInfoResponse {
	mode := "jwt"
	if config.THA != nil && config.THA.Enabled {
		mode = "header"
	}
	oidcEnabled := mode == "jwt" && config.OIDC.Enabled && config.OIDC.IssuerURL != "" && config.OIDC.ClientID != ""
	resp := &AuthInfoResponse{
		Mode:                  mode,
		Username:              username,
		Member:                s.MemberInspect(username),
		OIDCEnabled:           oidcEnabled,
		PasskeyEnabled:        s.PasskeyEnabled(),
		PasswordDisabled:      mode == "jwt" && config.Password.Disabled,
		PasswordMinLength:     config.Password.MinLength,
	}
	if oidcEnabled {
		resp.OIDCBtnLabel = config.OIDC.LoginLabel
	}
	return resp
}

// ─── 登录与 Token 签发 ──────────────

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
	TOTPCode string `json:"totpCode"`                    // TOTP 验证码（启用二步验证时必填）
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token             string `json:"token,omitempty"`             // JWT Token（二步验证时为空）
	Username          string `json:"username"`                    // 用户名
	TwoFactorRequired bool   `json:"twoFactorRequired,omitempty"` // 是否需要二步验证
}

// Login 校验用户名密码并签发 JWT Token
func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	if config.Password.Disabled {
		return nil, fmt.Errorf("密码登录已禁用")
	}
	member, exists := config.Members[req.Username]
	if !exists || !secure.BcryptVerify(req.Password, member.Password) {
		logman.Warn("Login failed", "username", req.Username)
		return nil, fmt.Errorf("invalid credentials")
	}
	if s.TOTPEnabled(member) {
		if req.TOTPCode == "" {
			return &LoginResponse{Username: req.Username, TwoFactorRequired: true}, nil
		}
		if !s.TOTPValidate(member.TwoFactor.TOTP.Secret, req.TOTPCode) {
			logman.Warn("TOTP login failed", "username", req.Username)
			return nil, fmt.Errorf("验证码无效")
		}
	}
	resp, err := s.IssueLoginToken(req.Username)
	if err != nil {
		return nil, err
	}
	logman.Info("User logged in", "username", req.Username)
	return resp, nil
}

// IssueLoginToken 为已存在成员签发登录 JWT Token
func (s *Service) IssueLoginToken(username string) (*LoginResponse, error) {
	tokenStr, err := s.signJWT(username, jwt.MapClaims{
		"exp": time.Now().Add(time.Duration(config.Server.JWTExpiration) * time.Second).Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &LoginResponse{Token: tokenStr, Username: username}, nil
}

// CreateApiTokenRequest 创建 API Token 请求
type CreateApiTokenRequest struct {
	Name      string `json:"name"`      // 令牌名称（用于标识）
	ExpiresIn int64  `json:"expiresIn"` // 过期时间（秒），0 表示永不过期
}

// CreateApiTokenResponse 创建 API Token 响应
type CreateApiTokenResponse struct {
	Token string `json:"token"` // 新创建的 API Token（仅创建时返回明文）
	Name  string `json:"name"`  // 令牌名称
}

// ApiTokenCreate 为已认证用户创建长效 API Token
func (s *Service) ApiTokenCreate(username string, req CreateApiTokenRequest) (*CreateApiTokenResponse, error) {
	extra := jwt.MapClaims{
		"type": "api",
		"name": req.Name,
	}
	if req.ExpiresIn > 0 {
		extra["exp"] = time.Now().Add(time.Duration(req.ExpiresIn) * time.Second).Unix()
	}
	tokenStr, err := s.signJWT(username, extra)
	if err != nil {
		return nil, err
	}
	logman.Info("API token created", "username", username, "name", req.Name)
	return &CreateApiTokenResponse{Token: tokenStr, Name: req.Name}, nil
}

// ─── JWT 认证 ──────────────

// JWTCheck 解析 JWT 并返回用户名；失败时返回空用户名和具体错误原因。
func (s *Service) JWTCheck(c *gin.Context) (username, errMsg string) {
	tokenStr := s.extractJWT(c)
	if tokenStr == "" {
		return "", "未提供认证令牌"
	}
	username = s.JWTUsername(c)
	if username == "" {
		return "", "认证令牌无效"
	}
	return username, ""
}

// JWTUsername 从 JWT 中解析并返回有效用户名；无效时返回空字符串
func (s *Service) JWTUsername(c *gin.Context) string {
	tokenStr := s.extractJWT(c)
	if tokenStr == "" {
		return ""
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Server.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}

	sub, _ := claims["sub"].(string)
	member, exists := config.Members[sub]
	if !exists {
		return ""
	}

	// 校验密码 hash 后 8 位（修改密码后自动失效）
	pwd, _ := claims["pwd"].(string)
	if pwd != "" && len(member.Password) >= 8 && pwd != member.Password[len(member.Password)-8:] {
		return ""
	}

	return sub
}

// ─── 代理 Header 登录 ────────────────

// HeaderTokenCheck 从上游代理传入的 Header 读取用户名；失败时返回空用户名和具体错误原因。
func (s *Service) HeaderTokenCheck(c *gin.Context) (username, errMsg string) {
	if !s.headerSourceTrusted(c) {
		return "", "代理 Header 来源不可信"
	}

	raw := c.GetHeader(config.THA.HeaderName)
	if raw == "" {
		return "", "代理 Header 缺失"
	}
	username = s.HeaderUsernameExtract(c)
	if username == "" {
		return "", "用户不存在"
	}
	return username, ""
}

// HeaderUsernameExtract 从代理 Header 中读取并验证用户名
func (s *Service) HeaderUsernameExtract(c *gin.Context) string {
	username := c.GetHeader(config.THA.HeaderName)
	if username == "" {
		return ""
	}
	if _, exists := config.Members[username]; !exists {
		return ""
	}
	return username
}

// ─── 内部方法 ───

// signJWT 为指定用户签发 JWT，extra 中的 claims 会合并到标准 claims 中
func (s *Service) signJWT(username string, extra jwt.MapClaims) (string, error) {
	member, exists := config.Members[username]
	if !exists {
		return "", fmt.Errorf("用户不存在")
	}

	// 密码 hash 后 8 位作为校验，修改密码后 token 自动失效
	// bcrypt hash 前 7 位是固定格式（如 $2a$10$），后 8 位会随密码重置而变化
	pwd := ""
	if len(member.Password) >= 8 {
		pwd = member.Password[len(member.Password)-8:]
	}

	claims := jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"pwd": pwd,
	}
	for k, v := range extra {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.Server.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("token 生成失败: %w", err)
	}
	return tokenStr, nil
}

// extractJWT 从 Authorization Header 或（路由标记了 QueryToken 时）query ?token= 中提取原始 JWT 字符串。
func (s *Service) extractJWT(c *gin.Context) string {
	tokenStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if tokenStr != "" {
		return tokenStr
	}
	// WebSocket 及标记了 QueryToken 的路由（SSE、文件预览下载）允许 query token
	if c.GetHeader("Upgrade") == "websocket" {
		return c.Query("token")
	}
	if v, exists := c.Get("routeQueryToken"); exists && v.(bool) {
		return c.Query("token")
	}
	return ""
}

func (s *Service) headerSourceTrusted(c *gin.Context) bool {
	// 未配置 TrustedCIDRs 时，向后兼容：不做来源限制
	if config.THA == nil || len(config.THA.TrustedCIDRs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, cidr := range config.THA.TrustedCIDRs {
		if trustedIP := net.ParseIP(cidr); trustedIP != nil && trustedIP.Equal(ip) {
			return true
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet.Contains(ip) {
			return true
		}
	}
	return false
}
