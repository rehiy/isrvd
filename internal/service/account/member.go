package account

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"

	"isrvd/config"
)

// 哨兵错误，供 handler 层进行错误类型判断
var (
	ErrMemberNotFound   = errors.New("成员不存在")
	ErrMemberExists     = errors.New("用户名已存在")
	ErrInvalidRequest   = errors.New("用户名不能为空")
	ErrFounderProtected = errors.New("创始人不可修改或删除")
	ErrPasskeyNotFound  = errors.New("凭证不存在")
)

// ─── 成员查询 ──────────

// MemberInfo 成员信息（不包含密码明文）
type MemberInfo struct {
	Username      string                  `json:"username"`            // 用户名（唯一标识）
	HomeDirectory string                  `json:"homeDirectory"`       // 家目录（绝对路径）
	Founder       bool                    `json:"founder"`             // 是否为创始人（不可删除/修改）
	Description   string                  `json:"description"`         // 成员描述
	Permissions   []string                `json:"permissions"`         // 权限列表
	TwoFactor     *config.TwoFactorConfig `json:"twoFactor,omitempty"` // 二步验证配置
}

// MemberInspect 获取单个成员信息
func (s *Service) MemberInspect(username string) *MemberInfo {
	m, exists := config.Members[username]
	if !exists {
		return nil
	}
	return s.memberInfoBuild(m)
}

// MemberList 列出所有成员
func (s *Service) MemberList() []*MemberInfo {
	list := make([]*MemberInfo, 0, len(config.Members))
	for _, m := range config.Members {
		list = append(list, s.memberInfoBuild(m))
	}
	return list
}

// ─── 成员创建/更新/删除 ──

// MemberUpsertRequest 成员新建/更新请求
type MemberUpsertRequest struct {
	Username      string   `json:"username"`      // 用户名（创建时必填，更新时从 URL 读取）
	Password      string   `json:"password"`      // 密码（创建时必填，更新时为空则保留原密码）
	HomeDirectory string   `json:"homeDirectory"` // 家目录（绝对路径或基于 RootDirectory 的相对路径）
	Description   string   `json:"description"`   // 成员描述
	Permissions   []string `json:"permissions"`   // 权限列表
}

// MemberCreate 新建成员
func (s *Service) MemberCreate(req MemberUpsertRequest) error {
	if req.Username == "" {
		return ErrInvalidRequest
	}
	if _, exists := config.Members[req.Username]; exists {
		return ErrMemberExists
	}

	home, err := s.homeDirEnsure(req.HomeDirectory, req.Username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}

	// 对密码进行 bcrypt 加密
	hashedPassword, err := secure.BcryptHash(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	config.Members[req.Username] = &config.MemberConfig{
		Username:      req.Username,
		Password:      hashedPassword,
		HomeDirectory: home,
		Description:   req.Description,
		Permissions:   req.Permissions,
	}
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member created", "username", req.Username)
	return nil
}

// MemberUpdate 更新成员
func (s *Service) MemberUpdate(username string, req MemberUpsertRequest) error {
	member, exists := config.Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	// 创始人不可修改
	if member.Founder {
		return ErrFounderProtected
	}

	home, err := s.homeDirEnsure(req.HomeDirectory, username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}

	// 密码为空时 Hash 返回空，保持原密码不变
	hashedPassword, err := secure.BcryptHash(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	if hashedPassword != "" {
		member.Password = hashedPassword
	}

	member.HomeDirectory = home
	member.Description = req.Description
	member.Permissions = req.Permissions

	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member updated", "username", username)
	return nil
}

// MemberDelete 删除成员
func (s *Service) MemberDelete(username string) error {
	member, exists := config.Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	// 创始人不可删除
	if member.Founder {
		return ErrFounderProtected
	}
	delete(config.Members, username)
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member deleted", "username", username)
	return nil
}

// ─── 密码修改 ──────────

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`                    // 原密码
	NewPassword string `json:"newPassword" binding:"required"` // 新密码
}

// PasswordChange 修改当前用户密码
func (s *Service) PasswordChange(username string, req ChangePasswordRequest) error {
	member, exists := config.Members[username]
	if !exists {
		return ErrMemberNotFound
	}

	// 验证旧密码
	if req.OldPassword == "" {
		return fmt.Errorf("请输入原密码")
	}
	if !secure.BcryptVerify(req.OldPassword, member.Password) {
		return fmt.Errorf("原密码错误")
	}

	// 加密新密码
	hashedPassword, err := secure.BcryptHash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	member.Password = hashedPassword
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	logman.Info("Password changed", "username", username)
	return nil
}

// ─── 辅助函数 ──────────

// memberInfoBuild 从配置构建成员信息（确保权限不为 nil）
func (s *Service) memberInfoBuild(m *config.MemberConfig) *MemberInfo {
	perms := m.Permissions
	if perms == nil {
		perms = []string{}
	}
	return &MemberInfo{
		Username:      m.Username,
		HomeDirectory: m.HomeDirectory,
		Founder:       m.Founder,
		Description:   m.Description,
		Permissions:   perms,
		TwoFactor:     m.TwoFactor,
	}
}

// homeDirEnsure 生成并创建成员 home 目录（空值时使用基础目录 + 用户名）
func (s *Service) homeDirEnsure(home, username string) (string, error) {
	if home == "" {
		home = username
	}
	if !filepath.IsAbs(home) {
		home = filepath.Join(config.Server.RootDirectory, home)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		return "", err
	}
	return home, nil
}
