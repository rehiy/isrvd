package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"isrvd/server/config"
	"isrvd/server/model"
)

// 认证服务
type AuthService struct{}

// 认证服务实例
var AuthInstance *AuthService

// 创建认证服务实例
func GetAuthService() *AuthService {
	if AuthInstance == nil {
		AuthInstance = &AuthService{}
	}
	return AuthInstance
}

// 验证用户名和密码
func (as *AuthService) Login(req model.LoginRequest) (*model.LoginResponse, error) {
	member, exists := config.Members[req.Username]

	// 验证用户凭据
	if !exists || member.Password != req.Password {
		return nil, errors.New("invalid credentials")
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	// 签名 token
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// 返回 token
	return &model.LoginResponse{
		Username: req.Username,
		Token:    tokenString,
	}, nil
}
