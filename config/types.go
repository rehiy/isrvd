package config

import "time"

// 配置结构
type Config struct {
	Server      *ServerConfig      `yaml:"server"`      // 服务配置
	Password    *PasswordConfig    `yaml:"password"`    // 密码登录配置
	Passkey     *PasskeyConfig     `yaml:"passkey"`     // Passkey 登录配置
	OIDC        *OIDCConfig        `yaml:"oidc"`        // OIDC 登录配置
	THA         *THAConfig         `yaml:"tha"`         // 代理 Header 登录配置
	Agent       *AgentConfig       `yaml:"agent"`       // Agent LLM 配置
	Apisix      *ApisixConfig      `yaml:"apisix"`      // APISIX 配置
	Caddy       *CaddyConfig       `yaml:"caddy"`       // Caddy 配置
	Docker      *DockerConfig      `yaml:"docker"`      // Docker 配置
	Monitor     *MonitorConfig     `yaml:"monitor"`     // 监控配置
	Marketplace *MarketplaceConfig `yaml:"marketplace"` // 应用市场配置
	Links       []*LinkConfig      `yaml:"links"`       // 工具栏链接列表
	Members     []*MemberConfig    `yaml:"members"`     // 成员列表
}

// 服务器配置
type ServerConfig struct {
	ListenAddr       string   `yaml:"listenAddr" json:"listenAddr"`             // 监听地址（如 :8080）
	RootDirectory    string   `yaml:"rootDirectory" json:"rootDirectory"`       // 根目录路径
	MaxUploadSize    int64    `yaml:"maxUploadSize" json:"maxUploadSize"`       // 文件上传最大大小（字节），默认 100MB
	AllowedOrigins   []string `yaml:"allowedOrigins" json:"allowedOrigins"`     // 允许的 Origin 列表，支持通配符 *
	JWTSecret        string   `yaml:"jwtSecret" json:"jwtSecret,omitempty"`     // 写入时为空表示保留原值；响应时不返回
	JWTExpiration    int64    `yaml:"jwtExpiration" json:"jwtExpiration"`       // JWT 过期时间（秒），默认 86400
	OpenAPI          bool     `yaml:"openapi" json:"openapi"`                   // 是否对外提供 OpenAPI 文档（/openapi/），默认关闭
	Debug            bool     `yaml:"debug" json:"debug"`                       // 是否启用调试模式
}

// 密码登录配置
type PasswordConfig struct {
	Disabled  bool `yaml:"disabled" json:"disabled"`   // 是否禁用密码登录（禁用后仅允许 Passkey/OIDC/THA 登录）
	MinLength int  `yaml:"minLength" json:"minLength"` // 密码最小长度（默认 6）；创建成员和修改密码时后端同步校验
}

// 代理 Header 登录配置
type THAConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`           // 是否启用代理 Header 登录
	HeaderName   string   `yaml:"headerName" json:"headerName"`     // 上游代理传入登录用户名的 Header 名称
	TrustedCIDRs []string `yaml:"trustedCIDRs" json:"trustedCIDRs"` // 允许传入登录 Header 的代理来源 CIDR；为空时不限制来源
}

// OIDC 登陆配置
type OIDCConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`                     // 是否启用 OIDC 登录
	IssuerURL     string   `yaml:"issuerUrl" json:"issuerUrl"`                 // OIDC 提供者 Issuer 地址
	ClientID      string   `yaml:"clientId" json:"clientId"`                   // OIDC 客户端 ID
	ClientSecret  string   `yaml:"clientSecret" json:"clientSecret,omitempty"` // 写入时为空表示保留原值；响应时不返回
	RedirectURL   string   `yaml:"redirectUrl" json:"redirectUrl"`             // 回调地址
	UsernameClaim string   `yaml:"usernameClaim" json:"usernameClaim"`         // 从哪个 claim 提取用户名（如 sub、email、preferred_username）
	Scopes        []string `yaml:"scopes" json:"scopes"`                       // 请求的 Scope 列表
	LoginLabel    string   `yaml:"loginLabel" json:"loginLabel"`               // OIDC 登录按钮自定义名称，留空时使用默认文案
}

// Passkey 登录配置
type PasskeyConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`     // 是否启用 Passkey 登录
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
	Name        string `yaml:"name" json:"name"`                   // 仓库名称（用于显示）
	Description string `yaml:"description" json:"description"`     // 仓库描述（可选）
	URL         string `yaml:"url" json:"url"`                     // 仓库地址，如 registry.example.com
	Username    string `yaml:"username" json:"username"`           // 用户名（可选）
	Password    string `yaml:"password" json:"password,omitempty"` // 登录密码：GET 响应由 ConfigAll 脱敏清空；PUT 时为空表示按 url+username 匹配保留原值，非空则更新（修改 url/username 会导致匹配不到旧值，需重新填写密码）
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
	Username      string               `yaml:"username" json:"username"`                           // 用户名
	Password      string               `yaml:"password" json:"-"`                                  // 敏感字段不序列化到 JSON
	HomeDirectory string               `yaml:"homeDirectory" json:"homeDirectory"`                 // 主目录
	Passkeys      []*PasskeyCredential `yaml:"passkeys" json:"passkeys,omitempty"`                 // Passkey 凭证列表
	TwoFactor     *TwoFactorConfig     `yaml:"twoFactor,omitempty" json:"twoFactor,omitempty"`     // 二次验证配置
	Founder       bool                 `yaml:"founder" json:"founder"`                             // Founder 创始人标志，创始人拥有所有模块的完整权限
	Description   string               `yaml:"description,omitempty" json:"description,omitempty"` // Description 成员描述信息（可选）
	Permissions   []string             `yaml:"permissions,omitempty" json:"permissions,omitempty"` // Permissions 允许访问的路由列表，格式为 "METHOD /api/path"，如 "GET /api/docker/containers"
}

// TwoFactorConfig 存储用户二次验证配置
type TwoFactorConfig struct {
	TOTP *TOTPConfig `yaml:"totp,omitempty" json:"totp,omitempty"` // TOTP 二次验证配置
}

// TOTPConfig 存储用户 TOTP 二次验证配置
type TOTPConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"` // 是否已启用 TOTP
	Secret  string `yaml:"secret" json:"-"`        // 敏感字段不序列化到 JSON
}

// PasskeyCredential 存储用户的 Passkey 凭证信息
type PasskeyCredential struct {
	IDBase64        string    `yaml:"idBase64" json:"idBase64"`               // 凭证 ID（Base64 编码）
	PublicKeyBase64 string    `yaml:"publicKeyBase64" json:"publicKeyBase64"` // 公鑰（Base64 编码）
	AAGUIDBase64    string    `yaml:"aaguidBase64" json:"aaguidBase64"`       // 认证器 AAGUID
	SignCount       uint32    `yaml:"signCount" json:"signCount"`             // 初始签名计数（仅存储初始值，运行时由内存维护）
	BackupEligible  bool      `yaml:"backupEligible" json:"backupEligible"`   // 凭证是否支持跨设备备份（BE 标志，注册后不变）
	BackupState     bool      `yaml:"backupState" json:"backupState"`         // 凭证当前是否已备份（BS 标志，可变）
	DisplayName     string    `yaml:"displayName" json:"displayName"`         // 凭证显示名称
	AddedAt         time.Time `yaml:"addedAt" json:"addedAt"`                 // 添加时间
}
