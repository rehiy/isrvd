package services

import (
	"errors"

	"isrvd/internal/config"
	"isrvd/internal/models"
	"isrvd/pkg/auth"
)

// AuthService 认证服务
type AuthService struct{}

// NewAuthService 创建认证服务实例
func NewAuthService() *AuthService {
	return &AuthService{}
}

// Login 用户登录
func (as *AuthService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	cfg := config.GetGlobal()

	// 验证用户名和密码
	if password, exists := cfg.UserMap[req.Username]; exists && password == req.Password {
		token := auth.Manager.CreateToken(req.Username)
		return &models.LoginResponse{
			Token: token,
			User:  req.Username,
		}, nil
	}

	return nil, errors.New("invalid credentials")
}

// Logout 用户登出
func (as *AuthService) Logout(token string) {
	auth.Manager.DeleteToken(token)
}

// ValidateToken 验证令牌
func (as *AuthService) ValidateToken(token string) bool {
	return auth.Manager.ValidateToken(token)
}

// Global auth service instance
var AuthServiceInstance = NewAuthService()
