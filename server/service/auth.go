package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"isrvd/server/config"
)

// AuthService 认证服务
type AuthService struct{}

// AuthInstance 认证服务实例
var AuthInstance *AuthService

// GetAuthService 创建认证服务实例
func GetAuthService() *AuthService {
	if AuthInstance == nil {
		AuthInstance = &AuthService{}
	}
	return AuthInstance
}

// LoginResult 登录结果
type LoginResult struct {
	Token    string
	Username string
}

// Login 验证用户名和密码，返回 token
func (as *AuthService) Login(username, password string) (*LoginResult, error) {
	member, exists := config.Members[username]

	// 验证用户凭据
	if !exists || member.Password != password {
		return nil, errors.New("invalid credentials")
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	// 签名 token
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Username: username,
		Token:    tokenString,
	}, nil
}
