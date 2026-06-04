// Package account 账号与认证业务模块
//
// 提供用户认证、登录、成员管理、OIDC、Passkey 等功能。
// 所有业务方法封装在 Service 中，由 server 层统一调用。
package account

import (
	"fmt"
	"isrvd/config"
	"slices"
	"sync"
	"time"
)

// Service 账号业务服务
type Service struct {
	// OIDC 临时状态存储（state/loginCode 均短期有效，内存存储即可）
	oidcMu         sync.Mutex
	oidcStates     map[string]oidcState
	oidcLoginCodes map[string]oidcLoginCode
	oidcProvider   oidcProviderCache

	// Passkey 会话存储
	passkeyStore *passkeySessionStore
}

// NewService 创建账号业务服务
func NewService() *Service {
	s := &Service{
		oidcStates:     make(map[string]oidcState),
		oidcLoginCodes: make(map[string]oidcLoginCode),
		passkeyStore:   newPasskeySessionStore(),
	}
	// 后台定期清理过期的 OIDC 临时状态
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupOIDC()
		}
	}()
	return s
}

// PermCheck 校验用户是否有权访问指定路由（"METHOD /api/path"）。
// label 用于错误提示；返回 nil 表示有权限，否则返回描述错误原因的 error。
func (s *Service) PermCheck(username, label, method, path string) error {
	member, exists := config.Members[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}
	if member.Founder {
		return nil
	}
	routeKey := method + " " + path
	if slices.Contains(member.Permissions, routeKey) {
		return nil
	}
	if label == "" {
		label = routeKey
	}
	return fmt.Errorf("无 %s 访问权限", label)
}
