package service

import (
	"errors"
	"sync"
	"time"

	"isrvd/server/config"
	"isrvd/server/helper"
	"isrvd/server/model"
)

// 认证服务
type AuthService struct {
	mutex    sync.RWMutex
	sessions map[string]time.Time
}

// 认证服务实例
var AuthInstance *AuthService

// 创建认证服务实例
func GetAuthService() *AuthService {
	if AuthInstance == nil {
		AuthInstance = &AuthService{
			sessions: make(map[string]time.Time),
		}
		go AuthInstance.CleanupExpired()
	}
	return AuthInstance
}

// 用户登录
func (as *AuthService) Login(req model.LoginRequest) (*model.LoginResponse, error) {
	cfg := config.GetGlobal()

	// 验证用户名和密码
	if password, exists := cfg.UserMap[req.Username]; exists && password == req.Password {
		return &model.LoginResponse{
			Token: as.CreateToken(req.Username),
			User:  req.Username,
		}, nil
	}

	return nil, errors.New("invalid credentials")
}

// 创建令牌
func (as *AuthService) CreateToken(username string) string {
	token := helper.Md5sum(username + time.Now().String())
	as.mutex.Lock()
	as.sessions[token] = time.Now().Add(24 * time.Hour)
	as.mutex.Unlock()
	return token
}

// 删除令牌
func (as *AuthService) DeleteToken(token string) {
	as.mutex.Lock()
	delete(as.sessions, token)
	as.mutex.Unlock()
}

// 验证令牌
func (as *AuthService) ValidateToken(token string) bool {
	as.mutex.RLock()
	expiry, exists := as.sessions[token]
	as.mutex.RUnlock()

	if !exists {
		return false
	}

	if expiry.Before(time.Now()) {
		as.DeleteToken(token)
		return false
	}

	return true
}

// 清理过期的会话
func (as *AuthService) CleanupExpired() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		as.mutex.Lock()
		now := time.Now()
		for token, expiry := range as.sessions {
			if expiry.Before(now) {
				delete(as.sessions, token)
			}
		}
		as.mutex.Unlock()
	}
}
