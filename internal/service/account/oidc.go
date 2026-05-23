package account

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"
	"golang.org/x/oauth2"

	"isrvd/config"
)

// ─── 常量与内部类型 ───

const (
	oidcStateTTL     = 10 * time.Minute
	oidcLoginCodeTTL = 2 * time.Minute
	oidcProviderTTL  = 10 * time.Minute
)

// oidcState 保存授权请求的临时状态（state → nonce 映射）
type oidcState struct {
	Nonce     string
	ExpiresAt time.Time
}

// oidcLoginCode 保存验证通过后的一次性登录码（loginCode → username 映射）
type oidcLoginCode struct {
	Username  string
	ExpiresAt time.Time
}

// OIDCExchangeRequest OIDC 一次性登录码交换请求
type OIDCExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ─── Provider 缓存 ────

// oidcProviderCache 缓存 OIDC Provider，避免每次请求都重新拉取元数据
type oidcProviderCache struct {
	mu        sync.Mutex
	provider  *oidc.Provider
	issuerURL string
	expiresAt time.Time
}

func (c *oidcProviderCache) get(ctx context.Context, issuerURL string) (*oidc.Provider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.provider != nil && c.issuerURL == issuerURL && time.Now().Before(c.expiresAt) {
		return c.provider, nil
	}

	p, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC Provider 初始化失败: %w", err)
	}
	c.provider = p
	c.issuerURL = issuerURL
	c.expiresAt = time.Now().Add(oidcProviderTTL)
	return p, nil
}

// ─── 公开业务方法 ─────

// OIDCLoginURL 生成 OIDC 授权跳转地址
func (s *Service) OIDCLoginURL(c *gin.Context) (string, error) {
	_, oauthConfig, err := s.newOAuthConfig(c.Request.Context(), c)
	if err != nil {
		return "", err
	}

	state, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("生成 state 失败: %w", err)
	}
	nonce, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	s.oidcMu.Lock()
	s.oidcStates[state] = oidcState{Nonce: nonce, ExpiresAt: time.Now().Add(oidcStateTTL)}
	s.oidcMu.Unlock()

	return oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce)), nil
}

// OIDCCallback 校验 OIDC 回调，成功时返回一次性登录码
func (s *Service) OIDCCallback(c *gin.Context) (string, error) {
	if errText := c.Query("error"); errText != "" {
		logman.Warn("OIDC callback error from IdP", "error", errText,
			"error_description", c.Query("error_description"))
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// 先查询 state（不删除），等全流程成功后再消费，避免失败后无法重试
	nonce, ok := s.lookupOIDCState(state)
	if !ok {
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// 根据 state 获取 OAuth 配置
	provider, oauthConfig, err := s.newOAuthConfig(c.Request.Context(), c)
	if err != nil {
		return "", err
	}

	// 使用授权码换取 token
	token, err := oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		logman.Warn("OIDC code exchange failed", "error", err)
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// 从授权码中提取 ID Token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// 验证 id_token 签名和 claims
	idToken, err := provider.Verifier(&oidc.Config{ClientID: config.OIDC.ClientID}).Verify(c.Request.Context(), rawIDToken)
	if err != nil {
		logman.Warn("OIDC id_token verify failed", "error", err)
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// nonce 统一返回相同错误消息，避免 oracle 攻击
	if idToken.Nonce != nonce {
		logman.Warn("OIDC nonce mismatch")
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	// 提取用户名：优先从 id_token claims 读，回落到 UserInfo endpoint
	username, err := oidcUsername(c.Request.Context(), provider, oauthConfig.TokenSource(c.Request.Context(), token), idToken, config.OIDC.UsernameClaim)
	if err != nil {
		logman.Warn("OIDC username claim error", "error", err, "claim", config.OIDC.UsernameClaim)
		return "", fmt.Errorf("OIDC 登录失败，请重试")
	}

	if _, exists := config.Members[username]; !exists {
		logman.Warn("OIDC user not in members", "username", username)
		return "", fmt.Errorf("用户未配置，请联系管理员添加成员")
	}

	loginCode, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("生成登录码失败: %w", err)
	}

	// 全流程验证通过，消费 state（一次性使用）并保存登录码
	s.consumeOIDCState(state)
	s.oidcMu.Lock()
	s.oidcLoginCodes[loginCode] = oidcLoginCode{Username: username, ExpiresAt: time.Now().Add(oidcLoginCodeTTL)}
	s.oidcMu.Unlock()

	return loginCode, nil
}

// OIDCExchange 使用一次性登录码换取系统 JWT
func (s *Service) OIDCExchange(code string) (*LoginResponse, error) {
	username, ok := s.consumeOIDCLoginCode(code)
	if !ok {
		return nil, fmt.Errorf("登录码无效或已过期")
	}
	return s.IssueLoginToken(username)
}

// ─── 内部辅助方法 ─────

// newOAuthConfig 构建 oauth2.Config，供 OIDCLoginURL 和 OIDCCallback 共用
func (s *Service) newOAuthConfig(ctx context.Context, c *gin.Context) (*oidc.Provider, *oauth2.Config, error) {
	conf := config.OIDC
	if !conf.Enabled || conf.IssuerURL == "" || conf.ClientID == "" {
		return nil, nil, fmt.Errorf("OIDC 未启用或配置不完整")
	}
	provider, err := s.oidcProvider.get(ctx, conf.IssuerURL)
	if err != nil {
		return nil, nil, err
	}
	oauthConfig := &oauth2.Config{
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		RedirectURL:  oidcRedirectURL(c),
		Endpoint:     provider.Endpoint(),
		Scopes:       oidcScopes(conf.Scopes),
	}
	return provider, oauthConfig, nil
}

// lookupOIDCState 查询 state 对应的 nonce（只读，不删除）
func (s *Service) lookupOIDCState(state string) (string, bool) {
	if state == "" {
		return "", false
	}
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()
	stored, ok := s.oidcStates[state]
	if !ok {
		return "", false
	}
	if time.Now().After(stored.ExpiresAt) {
		delete(s.oidcStates, state)
		return "", false
	}
	return stored.Nonce, true
}

// consumeOIDCState 删除已使用的 state
func (s *Service) consumeOIDCState(state string) {
	if state == "" {
		return
	}
	s.oidcMu.Lock()
	delete(s.oidcStates, state)
	s.oidcMu.Unlock()
}

// consumeOIDCLoginCode 消费一次性登录码，返回对应用户名
func (s *Service) consumeOIDCLoginCode(code string) (string, bool) {
	if code == "" {
		return "", false
	}
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()
	stored, ok := s.oidcLoginCodes[code]
	if !ok {
		return "", false
	}
	delete(s.oidcLoginCodes, code)
	if time.Now().After(stored.ExpiresAt) {
		return "", false
	}
	return stored.Username, true
}

// cleanupOIDC 清理过期的 state 和 loginCode（由后台 goroutine 定期调用）
func (s *Service) cleanupOIDC() {
	s.oidcMu.Lock()
	defer s.oidcMu.Unlock()
	now := time.Now()
	for k, v := range s.oidcStates {
		if now.After(v.ExpiresAt) {
			delete(s.oidcStates, k)
		}
	}
	for k, v := range s.oidcLoginCodes {
		if now.After(v.ExpiresAt) {
			delete(s.oidcLoginCodes, k)
		}
	}
}

// ─── 纯函数辅助 ───────

// oidcUsername 从 id_token claims 或 UserInfo endpoint 中提取用户名
func oidcUsername(ctx context.Context, provider *oidc.Provider, tokenSource oauth2.TokenSource, idToken *oidc.IDToken, claimName string) (string, error) {
	claimName = strings.TrimSpace(claimName)
	if claimName == "" || claimName == "sub" {
		return idToken.Subject, nil
	}

	// 先尝试从 id_token 读取
	var idClaims map[string]any
	if err := idToken.Claims(&idClaims); err == nil {
		if v, _ := idClaims[claimName].(string); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
	}

	// 回落到 UserInfo endpoint
	userInfo, err := provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return "", fmt.Errorf("获取 UserInfo 失败: %w", err)
	}
	var uClaims map[string]any
	if err := userInfo.Claims(&uClaims); err != nil {
		return "", fmt.Errorf("UserInfo claims 解析失败")
	}
	value, _ := uClaims[claimName].(string)
	if value = strings.TrimSpace(value); value == "" {
		return "", fmt.Errorf("OIDC claim %s 为空", claimName)
	}
	return value, nil
}

// oidcScopes 规整 scopes 列表，确保包含 openid
func oidcScopes(scopes []string) []string {
	result := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}
	if !slices.Contains(result, "openid") {
		result = append([]string{"openid"}, result...)
	}
	return result
}

// oidcRedirectURL 返回 OIDC 回调地址。
// 优先使用配置中的固定地址；未配置时从请求 Host 动态生成（需在可信代理后运行）。
func oidcRedirectURL(c *gin.Context) string {
	if config.OIDC.RedirectURL != "" {
		return config.OIDC.RedirectURL
	}

	logman.Warn("OIDC redirectUrl not configured, generating from request headers. " +
		"Ensure X-Forwarded-* headers are set only by trusted proxies.")

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(strings.SplitN(c.GetHeader("X-Forwarded-Proto"), ",", 2)[0]); proto == "http" || proto == "https" {
		scheme = proto
	}

	host := c.Request.Host
	if fh := strings.TrimSpace(strings.SplitN(c.GetHeader("X-Forwarded-Host"), ",", 2)[0]); fh != "" {
		host = fh
	}

	return (&url.URL{Scheme: scheme, Host: host, Path: "/api/account/oidc/callback"}).String()
}

// randomToken 生成 32 字节的随机 URL-safe token
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
