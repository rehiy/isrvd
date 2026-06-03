package config

import "time"

// 配置结构
type Config struct {
	Server      *ServerConfig      `yaml:"server"`
	OIDC        *OIDCConfig        `yaml:"oidc"`
	Passkey     *PasskeyConfig     `yaml:"passkey"`
	Agent       *AgentConfig       `yaml:"agent"`
	Apisix      *ApisixConfig      `yaml:"apisix"`
	Caddy       *CaddyConfig       `yaml:"caddy"`
	Docker      *DockerConfig      `yaml:"docker"`
	Monitor     *MonitorConfig     `yaml:"monitor"`
	Marketplace *MarketplaceConfig `yaml:"marketplace"`
	Links       []*LinkConfig      `yaml:"links"`
	Members     []*MemberConfig    `yaml:"members"`
}

// 服务器配置
type ServerConfig struct {
	Debug             bool     `yaml:"debug" json:"debug"`
	ListenAddr        string   `yaml:"listenAddr" json:"listenAddr"`
	JWTSecret         string   `yaml:"jwtSecret" json:"jwtSecret,omitempty"`       // 写入时为空表示保留原值；响应时不返回
	JWTExpiration     int64    `yaml:"jwtExpiration" json:"jwtExpiration"`         // JWT 过期时间（秒），默认 86400
	ProxyHeaderName   string   `yaml:"proxyHeaderName" json:"proxyHeaderName"`     // 使用代理认证的请求头名称
	ProxyTrustedCIDRs []string `yaml:"proxyTrustedCIDRs" json:"proxyTrustedCIDRs"` // Header 代理认证可信来源 CIDR；为空时仅信任本机
	AllowedOrigins    []string `yaml:"allowedOrigins" json:"allowedOrigins"`       // 允许的 Origin 列表，支持通配符 *
	MaxUploadSize     int64    `yaml:"maxUploadSize" json:"maxUploadSize"`         // 文件上传最大大小（字节），默认 100MB
	RootDirectory     string   `yaml:"rootDirectory" json:"rootDirectory"`
}

// OIDC 配置
type OIDCConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	IssuerURL     string   `yaml:"issuerUrl" json:"issuerUrl"`
	ClientID      string   `yaml:"clientId" json:"clientId"`
	ClientSecret  string   `yaml:"clientSecret" json:"clientSecret,omitempty"` // 写入时为空表示保留原值；响应时不返回
	RedirectURL   string   `yaml:"redirectUrl" json:"redirectUrl"`
	UsernameClaim string   `yaml:"usernameClaim" json:"usernameClaim"`
	Scopes        []string `yaml:"scopes" json:"scopes"`
	LoginLabel    string   `yaml:"loginLabel" json:"loginLabel"` // OIDC 登录按钮自定义名称，留空时使用默认文案
}

// Passkey 配置
type PasskeyConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	RPName    string   `yaml:"rpName" json:"rpName"`       // Relying Party 名称
	RPID      string   `yaml:"rpId" json:"rpId"`           // Relying Party ID（通常是域名）
	RPOrigins []string `yaml:"rpOrigins" json:"rpOrigins"` // 允许的 Origin 列表
	Timeout   int      `yaml:"timeout" json:"timeout"`     // 超时时间（毫秒），默认 60000
}

// Agent LLM 配置
type AgentConfig struct {
	Model   string `yaml:"model" json:"model"`             // 模型名称
	BaseURL string `yaml:"baseUrl" json:"baseUrl"`         // LLM API 基础地址（OpenAI 兼容）
	APIKey  string `yaml:"apiKey" json:"apiKey,omitempty"` // 写入时为空表示保留原值；响应时不返回
}

// 监控配置
type MonitorConfig struct {
	Interval int `yaml:"interval" json:"interval"` // 采集间隔（秒），合法值：5/15/30/60；其他值均视为禁用
}

// Apisix 配置
type ApisixConfig struct {
	AdminURL string `yaml:"adminUrl" json:"adminUrl"`           // Apisix Admin API 地址
	AdminKey string `yaml:"adminKey" json:"adminKey,omitempty"` // 写入时为空表示保留原值；响应时不返回
}

// Caddy 配置
type CaddyConfig struct {
	AdminURL string `yaml:"adminUrl" json:"adminUrl"` // Caddy Admin API 地址，例如 http://127.0.0.1:2019
}

// Docker 配置
type DockerConfig struct {
	Host          string            `yaml:"host" json:"host"`                       // Docker 连接地址
	ContainerRoot string            `yaml:"containerRoot" json:"containerRoot"`     // 容器数据根目录
	Registries    []*DockerRegistry `yaml:"registries" json:"registries,omitempty"` // 镜像仓库配置列表
}

// 镜像仓库配置
type DockerRegistry struct {
	Name        string `yaml:"name" json:"name"`               // 仓库名称（用于显示）
	Description string `yaml:"description" json:"description"` // 仓库描述（可选）
	URL         string `yaml:"url" json:"url"`                 // 仓库地址，如 registry.example.com
	Username    string `yaml:"username" json:"username"`       // 用户名（可选）
	Password    string `yaml:"password" json:"-"`              // 敏感字段不序列化到 JSON；写入时为空表示保留原值
}

// 应用市场配置
type MarketplaceConfig struct {
	URL string `yaml:"url" json:"url"` // 应用市场站点地址，通过 iframe 嵌入
}

// 工具栏链接配置
type LinkConfig struct {
	Label string `yaml:"label" json:"label"` // 显示名称
	URL   string `yaml:"url" json:"url"`     // 链接地址
	Icon  string `yaml:"icon" json:"icon"`   // Font Awesome 图标类名（可选，如 fa-link）
}

// 成员配置
type MemberConfig struct {
	Username      string               `yaml:"username" json:"username"`
	Password      string               `yaml:"password" json:"-"` // 敏感字段不序列化到 JSON
	HomeDirectory string               `yaml:"homeDirectory" json:"homeDirectory"`
	Passkeys      []*PasskeyCredential `yaml:"passkeys" json:"passkeys,omitempty"` // Passkey 凭证列表
	// Founder 创始人标志，创始人拥有所有模块的完整权限
	Founder bool `yaml:"founder" json:"founder"`
	// Description 成员描述信息（可选）
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Permissions 允许访问的路由列表，格式为 "METHOD /api/path"，如 "GET /api/docker/containers"
	Permissions []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`
}

// PasskeyAuthenticator 认证器信息
type PasskeyAuthenticator struct {
	AAGUIDBase64 string `yaml:"aaguidBase64" json:"aaguidBase64"`
	SignCount    uint32 `yaml:"signCount" json:"signCount"`
	CloneWarning bool   `yaml:"cloneWarning" json:"cloneWarning"`
}

// PasskeyFlags 凭证标志位
type PasskeyFlags struct {
	UserPresent    bool `yaml:"userPresent" json:"userPresent"`
	UserVerified   bool `yaml:"userVerified" json:"userVerified"`
	BackupEligible bool `yaml:"backupEligible" json:"backupEligible"`
	BackupState    bool `yaml:"backupState" json:"backupState"`
}

// PasskeyCredential 存储用户的 Passkey 凭证信息
type PasskeyCredential struct {
	IDBase64        string               `yaml:"idBase64" json:"idBase64"`               // 凭证 ID（Base64 编码）
	PublicKeyBase64 string               `yaml:"publicKeyBase64" json:"publicKeyBase64"` // 公钥（Base64 编码）
	AttestationType string               `yaml:"attestationType" json:"attestationType"`
	Authenticator   PasskeyAuthenticator `yaml:"authenticator" json:"authenticator"`
	Flags           PasskeyFlags         `yaml:"flags" json:"flags"`
	DisplayName     string               `yaml:"displayName" json:"displayName"`
	AddedAt         time.Time            `yaml:"addedAt" json:"addedAt"`
}
