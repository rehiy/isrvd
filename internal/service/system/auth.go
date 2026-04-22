// Package system 认证与登录业务逻辑
package system

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
)

// AuthInfoResponse 认证模式及当前用户信息
type AuthInfoResponse struct {
	Mode     string      `json:"mode"`
	Username string      `json:"username,omitempty"`
	Member   *MemberInfo `json:"member,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// AuthService 认证业务服务
type AuthService struct{}

// NewAuthService 创建认证业务服务
func NewAuthService() *AuthService {
	return &AuthService{}
}

// GetAuthInfo 返回当前认证模式及已登录用户信息
// header 模式下从代理 Header 读取用户名；jwt 模式下 username 由调用方从 token 中解析传入
func (s *AuthService) GetAuthInfo(username string) *AuthInfoResponse {
	mode := "jwt"
	if config.ProxyHeaderName != "" {
		mode = "header"
	}
	return &AuthInfoResponse{
		Mode:     mode,
		Username: username,
		Member:   s.getMember(username),
	}
}

// getMember 获取用户信息，不存在时返回 nil
func (s *AuthService) getMember(username string) *MemberInfo {
	if username == "" {
		return nil
	}
	m, exists := config.Members[username]
	if !exists {
		return nil
	}
	perms := m.Permissions
	if perms == nil {
		perms = map[string]string{}
	}
	return &MemberInfo{
		Username:      m.Username,
		HomeDirectory: m.HomeDirectory,
		AllowTerminal: m.AllowTerminal,
		PasswordSet:   m.Password != "",
		IsPrimary:     m.Username == config.PrimaryMember,
		Permissions:   perms,
	}
}

// Login 校验用户名密码并签发 JWT Token
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	member, exists := config.Members[req.Username]
	if !exists || member.Password != req.Password {
		logman.Warn("Login failed", "username", req.Username)
		return nil, fmt.Errorf("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("token 生成失败: %w", err)
	}

	logman.Info("User logged in", "username", req.Username)
	return &LoginResponse{Token: tokenString, Username: req.Username}, nil
}
