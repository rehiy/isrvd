package service

import (
	"errors"

	"isrvd/server/config"
	"isrvd/server/helper"
	"isrvd/server/model"
)

// 认证服务
type AuthService struct {
	session *helper.Session
}

// 创建认证服务实例
func NewAuthService() *AuthService {
	return &AuthService{
		session: helper.NewSession(),
	}
}

// 用户登录
func (as *AuthService) Login(req model.LoginRequest) (*model.LoginResponse, error) {
	cfg := config.GetGlobal()

	// 验证用户名和密码
	if password, exists := cfg.UserMap[req.Username]; exists && password == req.Password {
		token := as.session.CreateToken(req.Username)
		return &model.LoginResponse{
			Token: token,
			User:  req.Username,
		}, nil
	}

	return nil, errors.New("invalid credentials")
}

// 用户登出
func (as *AuthService) Logout(token string) {
	as.session.DeleteToken(token)
}

// 验证令牌
func (as *AuthService) ValidateToken(token string) bool {
	return as.session.ValidateToken(token)
}
