package account

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"

	"isrvd/server/config"
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
	m, exists := config.Current().Members[username]
	if !exists {
		return nil
	}
	return s.memberInfoBuild(m)
}

// MemberList 列出所有成员
func (s *Service) MemberList() []*MemberInfo {
	list := make([]*MemberInfo, 0, len(config.Current().Members))
	for _, m := range config.Current().Members {
		list = append(list, s.memberInfoBuild(m))
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Username < list[j].Username })
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
	s.memberMu.Lock()
	defer s.memberMu.Unlock()

	if req.Username == "" {
		return ErrInvalidRequest
	}
	snapshot := config.Current()
	if _, exists := snapshot.Members[req.Username]; exists {
		return ErrMemberExists
	}
	if len(req.Password) < snapshot.Password.MinLength {
		return fmt.Errorf("密码长度不能少于 %d 位", snapshot.Password.MinLength)
	}
	hashedPassword, err := secure.BcryptHash(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	home, homeCreated, err := s.homeDirEnsure(req.HomeDirectory, req.Username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}
	if err := config.Update(func(draft *config.Snapshot) error {
		if _, exists := draft.Members[req.Username]; exists {
			return ErrMemberExists
		}
		if len(req.Password) < draft.Password.MinLength {
			return fmt.Errorf("密码长度不能少于 %d 位", draft.Password.MinLength)
		}
		draft.Members[req.Username] = &config.MemberConfig{
			Username:      req.Username,
			Password:      hashedPassword,
			HomeDirectory: home,
			Description:   req.Description,
			Permissions:   append([]string(nil), req.Permissions...),
		}
		return nil
	}); err != nil {
		if homeCreated {
			if removeErr := os.Remove(home); removeErr != nil && !os.IsNotExist(removeErr) {
				logman.Warn("Rollback member home failed", "home", home, "error", removeErr)
			}
		}
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member created", "username", req.Username)
	return nil
}

// MemberUpdate 更新成员
func (s *Service) MemberUpdate(username string, req MemberUpsertRequest) error {
	s.memberMu.Lock()
	defer s.memberMu.Unlock()

	snapshot := config.Current()
	member, exists := snapshot.Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	if member.Founder {
		return ErrFounderProtected
	}
	if req.Password != "" && len(req.Password) < snapshot.Password.MinLength {
		return fmt.Errorf("密码长度不能少于 %d 位", snapshot.Password.MinLength)
	}
	hashedPassword, err := secure.BcryptHash(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	home, homeCreated, err := s.homeDirEnsure(req.HomeDirectory, username)
	if err != nil {
		return fmt.Errorf("创建 home 目录失败: %w", err)
	}
	if err := config.Update(func(draft *config.Snapshot) error {
		member, exists := draft.Members[username]
		if !exists {
			return ErrMemberNotFound
		}
		if member.Founder {
			return ErrFounderProtected
		}
		if hashedPassword != "" {
			if len(req.Password) < draft.Password.MinLength {
				return fmt.Errorf("密码长度不能少于 %d 位", draft.Password.MinLength)
			}
			member.Password = hashedPassword
		}
		member.HomeDirectory = home
		member.Description = req.Description
		member.Permissions = append([]string(nil), req.Permissions...)
		return nil
	}); err != nil {
		if homeCreated {
			if removeErr := os.Remove(home); removeErr != nil && !os.IsNotExist(removeErr) {
				logman.Warn("Rollback member home failed", "home", home, "error", removeErr)
			}
		}
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("Member updated", "username", username)
	return nil
}

// MemberDelete 删除成员
func (s *Service) MemberDelete(username string) error {
	if err := config.Update(func(draft *config.Snapshot) error {
		member, exists := draft.Members[username]
		if !exists {
			return ErrMemberNotFound
		}
		if member.Founder {
			return ErrFounderProtected
		}
		delete(draft.Members, username)
		return nil
	}); err != nil {
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
	member, exists := config.Current().Members[username]
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
	if minLen := config.Current().Password.MinLength; len(req.NewPassword) < minLen {
		return fmt.Errorf("密码长度不能少于 %d 位", minLen)
	}

	// 加密新密码
	hashedPassword, err := secure.BcryptHash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := config.Update(func(draft *config.Snapshot) error {
		currentMember, exists := draft.Members[username]
		if !exists {
			return ErrMemberNotFound
		}
		if !secure.BcryptVerify(req.OldPassword, currentMember.Password) {
			return fmt.Errorf("原密码错误")
		}
		if len(req.NewPassword) < draft.Password.MinLength {
			return fmt.Errorf("密码长度不能少于 %d 位", draft.Password.MinLength)
		}
		currentMember.Password = hashedPassword
		return nil
	}); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	logman.Info("Password changed", "username", username)
	return nil
}

// ─── 内部方法 ───

// memberInfoBuild 从配置构建成员信息（确保权限不为 nil）
func (s *Service) memberInfoBuild(m *config.MemberConfig) *MemberInfo {
	permissions := append([]string(nil), m.Permissions...)
	if permissions == nil {
		permissions = []string{}
	}
	var twoFactor *config.TwoFactorConfig
	if m.TwoFactor != nil {
		twoFactor = &config.TwoFactorConfig{}
		if m.TwoFactor.TOTP != nil {
			totp := *m.TwoFactor.TOTP
			twoFactor.TOTP = &totp
		}
	}
	return &MemberInfo{
		Username:      m.Username,
		HomeDirectory: m.HomeDirectory,
		Founder:       m.Founder,
		Description:   m.Description,
		Permissions:   permissions,
		TwoFactor:     twoFactor,
	}
}

// homeDirEnsure 生成并创建成员 home 目录，并报告最终目录是否由本次调用原子创建。
func (s *Service) homeDirEnsure(home, username string) (string, bool, error) {
	if home == "" {
		home = username
	}
	if !filepath.IsAbs(home) {
		home = filepath.Join(config.Current().Server.RootDirectory, home)
	}
	if err := os.MkdirAll(filepath.Dir(home), 0755); err != nil {
		return "", false, err
	}
	if err := os.Mkdir(home, 0755); err == nil {
		return home, true, nil
	} else if !os.IsExist(err) {
		return "", false, err
	}
	info, err := os.Stat(home)
	if err != nil {
		return "", false, err
	}
	if !info.IsDir() {
		return "", false, fmt.Errorf("home 路径不是目录: %s", home)
	}
	return home, false, nil
}
