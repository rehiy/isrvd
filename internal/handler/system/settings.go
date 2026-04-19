// Package system 系统配置查询与修改
package system

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/config"
	"isrvd/internal/helper"
)

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// ServerSettings 服务器配置
// 敏感字段：GET 时值为空，仅通过 *Set 标志位告知是否已设置；PUT 时值为空表示保留原值
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

// AllSettings 全部配置聚合
type AllSettings struct {
	Server      *ServerSettings      `json:"server"`
	Agent       *AgentSettings       `json:"agent"`
	Apisix      *ApisixSettings      `json:"apisix"`
	Docker      *DockerSettings      `json:"docker"`
	Marketplace *MarketplaceSettings `json:"marketplace"`
}

// UpdateAllRequest 全量更新请求（各分区均可选，nil 表示该分区不更新）
type UpdateAllRequest struct {
	Server      *ServerSettings      `json:"server"`
	Agent       *AgentSettings       `json:"agent"`
	Apisix      *ApisixSettings      `json:"apisix"`
	Docker      *DockerSettings      `json:"docker"`
	Marketplace *MarketplaceSettings `json:"marketplace"`
}

// pickSecret 新值为空时保留原值，否则用新值
func pickSecret(newVal, oldVal string) string {
	if newVal == "" {
		return oldVal
	}
	return newVal
}

// GetAll 获取全部配置
func (h *SettingsHandler) GetAll(c *gin.Context) {
	helper.RespondSuccess(c, "ok", &AllSettings{
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
	})
}

// UpdateAll 一次性更新全部配置（任何 nil 分区将跳过）
func (h *SettingsHandler) UpdateAll(c *gin.Context) {
	var req UpdateAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
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
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "全部配置已保存，部分项需重启生效", nil)
}
