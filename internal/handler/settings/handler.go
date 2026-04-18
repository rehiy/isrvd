// Package settings 提供系统配置查询与修改接口
package settings

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

// AllSettings 全部配置聚合
type AllSettings struct {
	Server *ServerSettings `json:"server"`
	Agent  *AgentSettings  `json:"agent"`
	Apisix *ApisixSettings `json:"apisix"`
	Docker *DockerSettings `json:"docker"`
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
	})
}

// UpdateServer 更新服务器配置
func (h *SettingsHandler) UpdateServer(c *gin.Context) {
	var req ServerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	config.Debug = req.Debug
	config.ListenAddr = req.ListenAddr
	config.JWTSecret = pickSecret(req.JWTSecret, config.JWTSecret)
	config.ProxyHeaderName = req.ProxyHeaderName
	config.RootDirectory = req.RootDirectory
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "服务器配置已保存，重启后完全生效", nil)
}

// UpdateAgent 更新 Agent LLM 配置
func (h *SettingsHandler) UpdateAgent(c *gin.Context) {
	var req AgentSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	config.Agent.Model = req.Model
	config.Agent.BaseURL = req.BaseURL
	config.Agent.APIKey = pickSecret(req.APIKey, config.Agent.APIKey)
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "Agent 配置已保存", nil)
}

// UpdateApisix 更新 Apisix 配置
func (h *SettingsHandler) UpdateApisix(c *gin.Context) {
	var req ApisixSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	config.Apisix.AdminURL = req.AdminURL
	config.Apisix.AdminKey = pickSecret(req.AdminKey, config.Apisix.AdminKey)
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "Apisix 配置已保存，重启后生效", nil)
}

// UpdateDocker 更新 Docker 配置
func (h *SettingsHandler) UpdateDocker(c *gin.Context) {
	var req DockerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	config.Docker.Host = req.Host
	config.Docker.ContainerRoot = req.ContainerRoot
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "Docker 配置已保存，重启后生效", nil)
}
