// Package system 系统配置查询与修改
package system

import (
	"fmt"

	"isrvd/config"
)

// ServerSettings 服务器配置
type ServerSettings struct {
	Debug           bool   `json:"debug"`
	ListenAddr      string `json:"listenAddr"`
	JWTSecret       string `json:"jwtSecret"`
	JWTSecretSet    bool   `json:"jwtSecretSet"`
	ProxyHeaderName string `json:"proxyHeaderName"`
	RootDirectory   string `json:"rootDirectory"`
}

// ApisixSettings Apisix 配置
type ApisixSettings struct {
	AdminURL    string `json:"adminUrl"`
	AdminKey    string `json:"adminKey"`
	AdminKeySet bool   `json:"adminKeySet"`
}

// AgentSettings Agent LLM 配置
type AgentSettings struct {
	Model     string `json:"model"`
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey"`
	APIKeySet bool   `json:"apiKeySet"`
}

// DockerSettings Docker 配置
type DockerSettings struct {
	Host          string `json:"host"`
	ContainerRoot string `json:"containerRoot"`
}

// MarketplaceSettings 应用市场配置
type MarketplaceSettings struct {
	URL string `json:"url"`
}

// EtcdSettings etcd 连接配置
type EtcdSettings struct {
	Endpoints    []string      `json:"endpoints"`
	Prefix       string        `json:"prefix"`
	Username     string        `json:"username"`
	UsernameSet  bool          `json:"usernameSet"`
	Password     string        `json:"password"`
	PasswordSet  bool          `json:"passwordSet"`
	TLSCertFile  string        `json:"tlsCertFile"`
	TLSKeyFile   string        `json:"tlsKeyFile"`
	TLSCAFile    string        `json:"tlsCaFile"`
}

// LinkConfig 工具栏链接
type LinkConfig struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
}

// AllSettings 全部配置聚合
type AllSettings struct {
	Server      *ServerSettings      `json:"server"`
	Agent       *AgentSettings       `json:"agent"`
	Apisix      *ApisixSettings      `json:"apisix"`
	Docker      *DockerSettings      `json:"docker"`
	Marketplace *MarketplaceSettings `json:"marketplace"`
	Etcd        *EtcdSettings        `json:"etcd"`
	Links       []*LinkConfig        `json:"links"`
}

// UpdateAllRequest 全量更新请求（各分区均可选，nil 表示该分区不更新）
type UpdateAllRequest struct {
	Server      *ServerSettings      `json:"server"`
	Agent       *AgentSettings       `json:"agent"`
	Apisix      *ApisixSettings      `json:"apisix"`
	Docker      *DockerSettings      `json:"docker"`
	Marketplace *MarketplaceSettings `json:"marketplace"`
	Etcd        *EtcdSettings        `json:"etcd"`
	Links       []*LinkConfig        `json:"links"`
}

// SettingsService 系统配置业务服务
type SettingsService struct{}

// NewSettingsService 创建系统配置业务服务
func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

// pickSecret 新值为空时保留原值，否则用新值
func pickSecret(newVal, oldVal string) string {
	if newVal == "" {
		return oldVal
	}
	return newVal
}

// GetAll 获取全部配置
func (s *SettingsService) GetAll() *AllSettings {
	return &AllSettings{
		Server: &ServerSettings{
			Debug:           config.Debug,
			ListenAddr:      config.ListenAddr,
			JWTSecretSet:    config.JWTSecret != "",
			ProxyHeaderName: config.ProxyHeaderName,
			RootDirectory:   config.RootDirectory,
		},
		Agent: &AgentSettings{
			Model:     config.Agent.Model,
			BaseURL:   config.Agent.BaseURL,
			APIKeySet: config.Agent.APIKey != "",
		},
		Apisix: &ApisixSettings{
			AdminURL:    config.Apisix.AdminURL,
			AdminKeySet: config.Apisix.AdminKey != "",
		},
		Docker: &DockerSettings{
			Host:          config.Docker.Host,
			ContainerRoot: config.Docker.ContainerRoot,
		},
		Marketplace: &MarketplaceSettings{
			URL: config.Marketplace.URL,
		},
		Etcd: &EtcdSettings{
			Endpoints:    config.Etcd.Endpoints,
			Prefix:       config.Etcd.Prefix,
			Username:      config.Etcd.Username,
			UsernameSet:   config.Etcd.Username != "",
			Password:      config.Etcd.Password,
			PasswordSet:   config.Etcd.Password != "",
			TLSCertFile:  config.Etcd.TLS.CertFile,
			TLSKeyFile:   config.Etcd.TLS.KeyFile,
			TLSCAFile:    config.Etcd.TLS.CAFile,
		},
		Links: func() []*LinkConfig {
			links := make([]*LinkConfig, 0, len(config.Links))
			for _, l := range config.Links {
				links = append(links, &LinkConfig{Label: l.Label, URL: l.URL, Icon: l.Icon})
			}
			return links
		}(),
	}
}

// UpdateAll 一次性更新全部配置（任何 nil 分区将跳过）
func (s *SettingsService) UpdateAll(req UpdateAllRequest) error {
	if req.Server != nil {
		config.Debug = req.Server.Debug
		config.ListenAddr = req.Server.ListenAddr
		config.JWTSecret = pickSecret(req.Server.JWTSecret, config.JWTSecret)
		config.ProxyHeaderName = req.Server.ProxyHeaderName
		config.RootDirectory = req.Server.RootDirectory
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
	if req.Docker != nil {
		config.Docker.Host = req.Docker.Host
		config.Docker.ContainerRoot = req.Docker.ContainerRoot
	}
	if req.Marketplace != nil {
		config.Marketplace.URL = req.Marketplace.URL
	}
	if req.Etcd != nil {
		config.Etcd.Endpoints = req.Etcd.Endpoints
		config.Etcd.Prefix = req.Etcd.Prefix
		config.Etcd.Username = pickSecret(req.Etcd.Username, config.Etcd.Username)
		config.Etcd.Password = pickSecret(req.Etcd.Password, config.Etcd.Password)
		if req.Etcd.TLSCertFile != "" || req.Etcd.TLSKeyFile != "" || req.Etcd.TLSCAFile != "" {
			if config.Etcd.TLS == nil {
				config.Etcd.TLS = &config.EtcdTLS{}
			}
			config.Etcd.TLS.CertFile = req.Etcd.TLSCertFile
			config.Etcd.TLS.KeyFile = req.Etcd.TLSKeyFile
			config.Etcd.TLS.CAFile = req.Etcd.TLSCAFile
		}
	}
	if req.Links != nil {
		links := make([]*config.LinkConfig, 0, len(req.Links))
		for _, l := range req.Links {
			links = append(links, &config.LinkConfig{Label: l.Label, URL: l.URL, Icon: l.Icon})
		}
		config.Links = links
	}
	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return nil
}
