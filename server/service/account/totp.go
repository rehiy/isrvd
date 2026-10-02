package account

import (
	"fmt"
	"strings"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"

	"isrvd/server/config"
)

const totpIssuer = "iSrvd"

// TwoFactorStatusResponse 二次验证状态
// 当前仅支持 TOTP。
type TwoFactorStatusResponse struct {
	Enabled bool `json:"enabled"` // 是否已启用两步验证
}

// TOTPBeginResponse 开始绑定 TOTP 的响应
// Secret 仅在绑定流程中返回一次，后续不会通过状态接口返回。
type TOTPBeginResponse struct {
	Secret string `json:"secret"` // TOTP 密钥（Base32 编码）
	URI    string `json:"uri"`    // otpauth:// URI（用于生成二维码）
}

// TOTPVerifyRequest TOTP 验证请求
type TOTPVerifyRequest struct {
	Code   string `json:"code" binding:"required"` // 认证器中的 6 位验证码
	Secret string `json:"secret"`                  // TOTP 密钥（启用时提交，禁用时可省略）
}

// TwoFactorStatus 查询当前用户二次验证状态
func (s *Service) TwoFactorStatus(username string) (*TwoFactorStatusResponse, error) {
	member, exists := config.Current().Members[username]
	if !exists {
		return nil, ErrMemberNotFound
	}
	return &TwoFactorStatusResponse{Enabled: s.TOTPEnabled(member)}, nil
}

// TOTPEnabled 判断成员是否已启用 TOTP 二次验证
func (s *Service) TOTPEnabled(member *config.MemberConfig) bool {
	return member != nil && member.TwoFactor != nil && member.TwoFactor.TOTP != nil && member.TwoFactor.TOTP.Enabled && member.TwoFactor.TOTP.Secret != ""
}

// TOTPBegin 开始绑定 TOTP，生成临时密钥和 otpauth URI
func (s *Service) TOTPBegin(username string) (*TOTPBeginResponse, error) {
	member, exists := config.Current().Members[username]
	if !exists {
		return nil, ErrMemberNotFound
	}
	if s.TOTPEnabled(member) {
		return nil, fmt.Errorf("TOTP 二次验证已启用")
	}

	secret, err := secure.TOTPSecretGenerate()
	if err != nil {
		return nil, err
	}
	uri := secure.TOTPURI(totpIssuer, username, secret)
	return &TOTPBeginResponse{Secret: secret, URI: uri}, nil
}

// TOTPEnable 完成 TOTP 绑定，验证通过后保存密钥并启用
func (s *Service) TOTPEnable(username string, req TOTPVerifyRequest) error {
	member, exists := config.Current().Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	if s.TOTPEnabled(member) {
		return fmt.Errorf("TOTP 二次验证已启用")
	}
	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		return fmt.Errorf("缺少 TOTP 密钥")
	}
	if !secure.TOTPValidate(secret, req.Code) {
		return fmt.Errorf("验证码无效")
	}

	if err := config.Update(func(draft *config.Snapshot) error {
		currentMember, exists := draft.Members[username]
		if !exists {
			return ErrMemberNotFound
		}
		if s.TOTPEnabled(currentMember) {
			return fmt.Errorf("TOTP 二次验证已启用")
		}
		if currentMember.TwoFactor == nil {
			currentMember.TwoFactor = &config.TwoFactorConfig{}
		}
		currentMember.TwoFactor.TOTP = &config.TOTPConfig{Enabled: true, Secret: secret}
		return nil
	}); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("TOTP enabled", "username", username)
	return nil
}

// TOTPDisable 禁用当前用户 TOTP 二次验证，需提供当前验证码
func (s *Service) TOTPDisable(username string, req TOTPVerifyRequest) error {
	member, exists := config.Current().Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	if !s.TOTPEnabled(member) {
		return fmt.Errorf("TOTP 二次验证未启用")
	}
	if !secure.TOTPValidate(member.TwoFactor.TOTP.Secret, req.Code) {
		return fmt.Errorf("验证码无效")
	}

	if err := config.Update(func(draft *config.Snapshot) error {
		currentMember, exists := draft.Members[username]
		if !exists {
			return ErrMemberNotFound
		}
		if !s.TOTPEnabled(currentMember) {
			return fmt.Errorf("TOTP 二次验证未启用")
		}
		if !secure.TOTPValidate(currentMember.TwoFactor.TOTP.Secret, req.Code) {
			return fmt.Errorf("验证码无效")
		}
		currentMember.TwoFactor.TOTP.Enabled = false
		currentMember.TwoFactor.TOTP.Secret = ""
		return nil
	}); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("TOTP disabled", "username", username)
	return nil
}
