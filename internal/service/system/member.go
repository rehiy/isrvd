// Package system 成员账号管理
package system

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rehiy/pango/logman"

	"isrvd/config"
)

// MemberInfo 成员信息（不包含密码明文）
type MemberInfo struct {
	Username      string            `json:"username"`
	HomeDirectory string            `json:"homeDirectory"`
	PasswordSet   bool              `json:"passwordSet"`
	IsPrimary     bool              `json:"isPrimary"`
	Permissions   map[string]string `json:"permissions"`
}

// MemberUpsertRequest 成员新建/更新请求
type MemberUpsertRequest struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	HomeDirectory string            `json:"homeDirectory"`
	Permissions   map[string]string `json:"permissions"`
}

// MemberService 成员账号业务服务
type MemberService struct{}

// NewMemberService 创建成员账号业务服务
func NewMemberService() *MemberService {
	return &MemberService{}
}

// GetMember 获取单个成员信息
func (s *MemberService) GetMember(username string) *MemberInfo {
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
		PasswordSet:   m.Password != "",
		IsPrimary:     m.Username == config.PrimaryMember,
		Permissions:   perms,
	}
}

// ListMembers 列出所有成员
func (s *MemberService) ListMembers() []*MemberInfo {
	list := make([]*MemberInfo, 0, len(config.Members))
	for _, m := range config.Members {
		perms := m.Permissions
		if perms == nil {
			perms = map[string]string{}
		}
		list = append(list, &MemberInfo{
			Username:      m.Username,
			HomeDirectory: m.HomeDirectory,
			PasswordSet:   m.Password != "",
			IsPrimary:     m.Username == config.PrimaryMember,
			Permissions:   perms,
		})
	}
	return list
}

// ensureHomeDir 生成并创建成员 home 目录（空值时使用基础目录 + 用户名）
func ensureHomeDir(home, username string) (string, error) {
	if home == "" {
		home = username
	}
	if !filepath.IsAbs(home) {
		home = filepath.Join(config.RootDirectory, home)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		return "", err
	}
	return home, nil
}

// CreateMember 新建成员
func (s *MemberService) CreateMember(req MemberUpsertRequest) error {
	if req.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if req.Password == "" {
		return fmt.Errorf("密码不能为空")
	}
	if _, exists := config.Members[req.Username]; exists {
		return fmt.Errorf("用户名已存在")
	}

	home, err := ensureHomeDir(req.HomeDirectory, req.Username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}

	config.Members[req.Username] = &config.MemberConfig{
		Username:      req.Username,
		Password:      req.Password,
		HomeDirectory: home,
		Permissions:   req.Permissions,
	}
	if config.PrimaryMember == "" {
		config.PrimaryMember = req.Username
	}
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member created", "username", req.Username)
	return nil
}

// UpdateMember 更新成员
func (s *MemberService) UpdateMember(username string, req MemberUpsertRequest) error {
	member, exists := config.Members[username]
	if !exists {
		return fmt.Errorf("成员不存在")
	}

	home, err := ensureHomeDir(req.HomeDirectory, username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}

	member.Password = pickSecret(req.Password, member.Password)
	member.HomeDirectory = home
	member.Permissions = req.Permissions

	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member updated", "username", username)
	return nil
}

// DeleteMember 删除成员
func (s *MemberService) DeleteMember(username string) error {
	if _, exists := config.Members[username]; !exists {
		return fmt.Errorf("成员不存在")
	}
	if username == config.PrimaryMember {
		return fmt.Errorf("主账号禁止删除")
	}
	delete(config.Members, username)
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member deleted", "username", username)
	return nil
}
