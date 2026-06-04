package account

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

const (
	totpIssuer = "iSrvd"
	totpPeriod = 30
	totpDigits = 6
)

// TwoFactorStatusResponse 二次验证状态
// 当前仅支持 TOTP。
type TwoFactorStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// TOTPBeginResponse 开始绑定 TOTP 的响应
// Secret 仅在绑定流程中返回一次，后续不会通过状态接口返回。
type TOTPBeginResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// TOTPVerifyRequest TOTP 验证请求
type TOTPVerifyRequest struct {
	Code   string `json:"code" binding:"required"`
	Secret string `json:"secret"`
}

// TwoFactorStatus 查询当前用户二次验证状态
func (s *Service) TwoFactorStatus(username string) (*TwoFactorStatusResponse, error) {
	member, exists := config.Members[username]
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
	member, exists := config.Members[username]
	if !exists {
		return nil, ErrMemberNotFound
	}
	if s.TOTPEnabled(member) {
		return nil, fmt.Errorf("TOTP 二次验证已启用")
	}

	secret, err := s.TOTPSecretGenerate()
	if err != nil {
		return nil, err
	}
	uri := s.TOTPURI(username, secret)
	return &TOTPBeginResponse{Secret: secret, URI: uri}, nil
}

// TOTPEnable 完成 TOTP 绑定，验证通过后保存密钥并启用
func (s *Service) TOTPEnable(username string, req TOTPVerifyRequest) error {
	member, exists := config.Members[username]
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
	if !s.TOTPValidate(secret, req.Code) {
		return fmt.Errorf("验证码无效")
	}

	if member.TwoFactor == nil {
		member.TwoFactor = &config.TwoFactorConfig{}
	}
	member.TwoFactor.TOTP = &config.TOTPConfig{
		Enabled: true,
		Secret:  secret,
	}
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("TOTP enabled", "username", username)
	return nil
}

// TOTPDisable 禁用当前用户 TOTP 二次验证，需提供当前验证码
func (s *Service) TOTPDisable(username string, req TOTPVerifyRequest) error {
	member, exists := config.Members[username]
	if !exists {
		return ErrMemberNotFound
	}
	if !s.TOTPEnabled(member) {
		return fmt.Errorf("TOTP 二次验证未启用")
	}
	if !s.TOTPValidate(member.TwoFactor.TOTP.Secret, req.Code) {
		return fmt.Errorf("验证码无效")
	}

	member.TwoFactor.TOTP.Enabled = false
	member.TwoFactor.TOTP.Secret = ""
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	logman.Info("TOTP disabled", "username", username)
	return nil
}

// TOTPSecretGenerate 生成 Base32 编码的 TOTP 密钥
func (s *Service) TOTPSecretGenerate() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 TOTP 密钥失败: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="), nil
}

// TOTPURI 生成可导入认证器 App 的 otpauth URI
func (s *Service) TOTPURI(username string, secret string) string {
	label := url.PathEscape(totpIssuer + ":" + username)
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", totpIssuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", fmt.Sprintf("%d", totpDigits))
	query.Set("period", fmt.Sprintf("%d", totpPeriod))
	return "otpauth://totp/" + label + "?" + query.Encode()
}

// TOTPValidate 验证 TOTP 验证码，允许前后一周期时间偏移
func (s *Service) TOTPValidate(secret string, code string) bool {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	code = strings.TrimSpace(code)
	if secret == "" || code == "" {
		return false
	}
	for _, offset := range []int64{-1, 0, 1} {
		if s.TOTPCode(secret, time.Now().Unix()/totpPeriod+offset) == code {
			return true
		}
	}
	return false
}

// TOTPCode 根据时间步生成 6 位验证码
func (s *Service) TOTPCode(secret string, counter int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", binaryCode%1000000)
}
