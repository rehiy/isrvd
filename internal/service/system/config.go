// Package system 系统配置查询与修改
package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"isrvd/config"
)

// AllConfig 全部配置聚合（请求/响应共用）。
// 作 GET 响应时：敏感字段已脱敏；作 PUT 请求时：nil 分区跳过更新，密钥为空保留原值。
type AllConfig struct {
	Server      *config.ServerConfig      `json:"server"`      // 服务配置（JWTSecret：响应脱敏 / 请求空保留）
	THA         *config.THAConfig         `json:"tha"`         // 代理 Header 认证配置
	OIDC        *config.OIDCConfig        `json:"oidc"`        // OIDC 配置（ClientSecret：响应脱敏 / 请求空保留）
	Passkey     *config.PasskeyConfig     `json:"passkey"`     // Passkey 认证配置
	Agent       *config.AgentConfig       `json:"agent"`       // Agent LLM 配置（APIKey：响应脱敏 / 请求空保留）
	Apisix      *config.ApisixConfig      `json:"apisix"`      // APISIX 配置（AdminKey：响应脱敏 / 请求空保留）
	Caddy       *config.CaddyConfig       `json:"caddy"`       // Caddy 配置
	Docker      *config.DockerConfig      `json:"docker"`      // Docker 配置（registry.Password：响应脱敏 / 请求空保留）
	Monitor     *config.MonitorConfig     `json:"monitor"`     // 监控配置
	Marketplace *config.MarketplaceConfig `json:"marketplace"` // 应用市场配置
	Links       []*config.LinkConfig      `json:"links"`       // 导航链接列表
}

// ConfigService 系统配置业务服务
type ConfigService struct{}

// NewConfigService 创建系统配置业务服务
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// ConfigAll 获取全部配置：深拷贝隔离全局 config 后清空敏感字段
func (s *ConfigService) ConfigAll() *AllConfig {
	src := &AllConfig{
		Server:      config.Server,
		THA:         config.THA,
		OIDC:        config.OIDC,
		Passkey:     config.Passkey,
		Agent:       config.Agent,
		Apisix:      config.Apisix,
		Caddy:       config.Caddy,
		Docker:      config.Docker,
		Monitor:     config.Monitor,
		Marketplace: config.Marketplace,
		Links:       config.Links,
	}
	dst, err := deepCopyJSON(src)
	if err != nil || dst == nil {
		return &AllConfig{}
	}
	// 脱敏：清空所有密钥/密码
	if dst.Server != nil {
		dst.Server.JWTSecret = ""
	}
	if dst.OIDC != nil {
		dst.OIDC.ClientSecret = ""
	}
	if dst.Agent != nil {
		dst.Agent.APIKey = ""
	}
	if dst.Apisix != nil {
		dst.Apisix.AdminKey = ""
	}
	if dst.Docker != nil {
		for _, r := range dst.Docker.Registries {
			if r != nil {
				r.Password = ""
			}
		}
	}
	return dst
}

// ConfigUpdate 一次性更新全部配置（任何 nil 分区将跳过）
func (s *ConfigService) ConfigUpdate(req AllConfig) error {
	if err := validateAuthConfig(req); err != nil {
		return err
	}
	if req.Server != nil {
		oldSecret := ""
		if config.Server != nil {
			oldSecret = config.Server.JWTSecret
		}
		req.Server.JWTSecret = pickSecret(req.Server.JWTSecret, oldSecret)
		config.Server = config.ServerNormalize(req.Server)
	}
	if req.THA != nil {
		config.THA = config.THANormalize(req.THA)
	}
	if req.OIDC != nil {
		oldSecret := ""
		if config.OIDC != nil {
			oldSecret = config.OIDC.ClientSecret
		}
		req.OIDC.ClientSecret = pickSecret(req.OIDC.ClientSecret, oldSecret)
		config.OIDC = config.OIDCNormalize(req.OIDC)
	}
	if req.Passkey != nil {
		config.Passkey = config.PasskeyNormalize(req.Passkey)
	}
	if req.Agent != nil {
		oldSecret := ""
		if config.Agent != nil {
			oldSecret = config.Agent.APIKey
		}
		req.Agent.APIKey = pickSecret(req.Agent.APIKey, oldSecret)
		config.Agent = req.Agent
	}
	if req.Apisix != nil {
		oldSecret := ""
		if config.Apisix != nil {
			oldSecret = config.Apisix.AdminKey
		}
		req.Apisix.AdminKey = pickSecret(req.Apisix.AdminKey, oldSecret)
		config.Apisix = req.Apisix
	}
	if req.Caddy != nil {
		config.Caddy = req.Caddy
	}
	if req.Docker != nil {
		// Registries 密码：空值按 url+username 匹配保留原值，非空则更新（改 url/username 会丢匹配，需重填）
		if config.Docker != nil {
			for _, reg := range req.Docker.Registries {
				if reg == nil || reg.Password != "" {
					continue
				}
				for _, old := range config.Docker.Registries {
					if old != nil && old.URL == reg.URL && old.Username == reg.Username {
						reg.Password = old.Password
						break
					}
				}
			}
		}
		config.Docker = req.Docker
	}
	if req.Monitor != nil {
		config.Monitor = config.MonitorNormalize(req.Monitor)
	}
	if req.Marketplace != nil {
		config.Marketplace = req.Marketplace
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

// deepCopyJSON 通过 JSON 序列化-反序列化深拷贝，结果与源对象无共享指针
func deepCopyJSON[T any](src T) (T, error) {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}
	err = json.Unmarshal(data, &dst)
	return dst, err
}

// pickSecret 新值为空（含纯空白）时保留原值，否则用裁剪首尾空白后的新值
func pickSecret(newVal, oldVal string) string {
	newVal = strings.TrimSpace(newVal)
	if newVal == "" {
		return oldVal
	}
	return newVal
}

func validateAuthConfig(req AllConfig) error {
	tha := config.THA
	if req.THA != nil {
		tha = config.THANormalize(req.THA)
	}
	oidc := config.OIDC
	if req.OIDC != nil {
		oidc = config.OIDCNormalize(req.OIDC)
	}
	if oidc == nil || !oidc.Only {
		return nil
	}
	if !oidc.Enabled || strings.TrimSpace(oidc.IssuerURL) == "" || strings.TrimSpace(oidc.ClientID) == "" {
		return fmt.Errorf("仅 OIDC 登录需要先启用 OIDC 并配置 issuerUrl 和 clientId")
	}
	if tha != nil && tha.Enabled {
		return fmt.Errorf("仅 OIDC 登录不能与代理 Header 登录同时启用")
	}
	return nil
}
