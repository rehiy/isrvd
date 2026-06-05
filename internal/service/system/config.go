// Package system 系统配置查询与修改
package system

import (
	"fmt"

	"isrvd/config"
)

// AllConfigResponse 全部配置聚合响应
type AllConfigResponse struct {
	Server      *config.ServerConfig      `json:"server"`
	THA         *config.THAConfig         `json:"tha"`
	OIDC        *config.OIDCConfig        `json:"oidc"`
	Passkey     *config.PasskeyConfig     `json:"passkey"`
	Agent       *config.AgentConfig       `json:"agent"`
	Apisix      *config.ApisixConfig      `json:"apisix"`
	Caddy       *config.CaddyConfig       `json:"caddy"`
	Docker      *config.DockerConfig      `json:"docker"`
	Monitor     *config.MonitorConfig     `json:"monitor"`
	Marketplace *config.MarketplaceConfig `json:"marketplace"`
	Links       []*config.LinkConfig      `json:"links"`
}

// UpdateAllConfigRequest 全量更新请求
type UpdateAllConfigRequest struct {
	Server      *config.ServerConfig      `json:"server"`
	THA         *config.THAConfig         `json:"tha"`
	OIDC        *config.OIDCConfig        `json:"oidc"`
	Passkey     *config.PasskeyConfig     `json:"passkey"`
	Agent       *config.AgentConfig       `json:"agent"`
	Apisix      *config.ApisixConfig      `json:"apisix"`
	Caddy       *config.CaddyConfig       `json:"caddy"`
	Docker      *config.DockerConfig      `json:"docker"`
	Monitor     *config.MonitorConfig     `json:"monitor"`
	Marketplace *config.MarketplaceConfig `json:"marketplace"`
	Links       []*config.LinkConfig      `json:"links"`
}

// ConfigService 系统配置业务服务
type ConfigService struct{}

// NewConfigService 创建系统配置业务服务
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// pickSecret 新值为空时保留原值，否则用新值
func pickSecret(newVal, oldVal string) string {
	if newVal == "" {
		return oldVal
	}
	return newVal
}

// ConfigAll 获取全部配置（显式拷贝，过滤敏感字段）
func (s *ConfigService) ConfigAll() *AllConfigResponse {
	srv := config.Server
	oidc := config.OIDC

	return &AllConfigResponse{
		Server: &config.ServerConfig{
			Debug:          srv.Debug,
			ListenAddr:     srv.ListenAddr,
			JWTExpiration:  srv.JWTExpiration,
			AllowedOrigins: srv.AllowedOrigins,
			MaxUploadSize:  srv.MaxUploadSize,
			RootDirectory:  srv.RootDirectory,
			// JWTSecret 不返回
		},
		THA: &config.THAConfig{
			Enabled:      config.THA.Enabled,
			HeaderName:   config.THA.HeaderName,
			TrustedCIDRs: config.THA.TrustedCIDRs,
		},
		OIDC: &config.OIDCConfig{
			Enabled:       oidc.Enabled,
			IssuerURL:     oidc.IssuerURL,
			ClientID:      oidc.ClientID,
			RedirectURL:   oidc.RedirectURL,
			UsernameClaim: oidc.UsernameClaim,
			Scopes:        oidc.Scopes,
			LoginLabel:    oidc.LoginLabel,
			// ClientSecret 不返回
		},
		Passkey: &config.PasskeyConfig{
			Enabled:   config.Passkey.Enabled,
			RPName:    config.Passkey.RPName,
			RPID:      config.Passkey.RPID,
			RPOrigins: config.Passkey.RPOrigins,
			Timeout:   config.Passkey.Timeout,
		},
		Agent: &config.AgentConfig{
			Model:   config.Agent.Model,
			BaseURL: config.Agent.BaseURL,
			// APIKey 不返回
		},
		Apisix: &config.ApisixConfig{
			AdminURL: config.Apisix.AdminURL,
			// AdminKey 不返回
		},
		Caddy: &config.CaddyConfig{
			AdminURL: config.Caddy.AdminURL,
		},
		Docker:      config.Docker,
		Monitor:     config.Monitor,
		Marketplace: config.Marketplace,
		Links:       config.Links,
	}
}

// ConfigUpdateAll 一次性更新全部配置（任何 nil 分区将跳过）
func (s *ConfigService) ConfigUpdateAll(req UpdateAllConfigRequest) error {
	if req.Server != nil {
		req.Server.JWTSecret = pickSecret(req.Server.JWTSecret, config.Server.JWTSecret)
		config.Server = config.ServerNormalize(req.Server)
	}
	if req.THA != nil {
		config.THA = config.THANormalize(req.THA)
	}
	if req.OIDC != nil {
		req.OIDC.ClientSecret = pickSecret(req.OIDC.ClientSecret, config.OIDC.ClientSecret)
		config.OIDC = config.OIDCNormalize(req.OIDC)
	}
	if req.Passkey != nil {
		config.Passkey = config.PasskeyNormalize(req.Passkey)
	}
	if req.Agent != nil {
		config.Agent.Model = req.Agent.Model
		config.Agent.BaseURL = req.Agent.BaseURL
		config.Agent.APIKey = pickSecret(req.Agent.APIKey, config.Agent.APIKey)
	}
	if req.Apisix != nil {
		config.Apisix.AdminURL = req.Apisix.AdminURL
		config.Apisix.AdminKey = pickSecret(req.Apisix.AdminKey, config.Apisix.AdminKey)
	}
	if req.Caddy != nil {
		config.Caddy.AdminURL = req.Caddy.AdminURL
	}
	if req.Docker != nil {
		config.Docker.Host = req.Docker.Host
		config.Docker.ContainerRoot = req.Docker.ContainerRoot
	}
	if req.Monitor != nil {
		config.Monitor = config.MonitorNormalize(req.Monitor)
	}
	if req.Marketplace != nil {
		config.Marketplace.URL = req.Marketplace.URL
	}
	if req.Links != nil {
		config.Links = req.Links
	}
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	// 触发重载，使新配置立即生效
	select {
	case config.ReloadCh <- struct{}{}:
	default:
	}
	return nil
}
