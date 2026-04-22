package config

// 配置结构
type Config struct {
	Server      *Server            `yaml:"server"`
	Agent       *AgentConfig       `yaml:"agent"`
	Apisix      *ApisixConfig      `yaml:"apisix"`
	Docker      *DockerConfig      `yaml:"docker"`
	Marketplace *MarketplaceConfig `yaml:"marketplace"`
	Links       []*LinkConfig      `yaml:"links"`
	Members     []*MemberConfig    `yaml:"members"`
}

// 服务器配置
type Server struct {
	Debug           bool   `yaml:"debug"`
	ListenAddr      string `yaml:"listenAddr"`
	JWTSecret       string `yaml:"jwtSecret"`
	ProxyHeaderName string `yaml:"proxyHeaderName"`
	RootDirectory   string `yaml:"rootDirectory"`
}

// Agent LLM 配置
type AgentConfig struct {
	Model   string `yaml:"model"`   // 模型名称
	BaseURL string `yaml:"baseUrl"` // LLM API 基础地址（OpenAI 兼容）
	APIKey  string `yaml:"apiKey"`  // API 密钥（不回流到前端）
}

// Apisix 配置
type ApisixConfig struct {
	AdminURL string `yaml:"adminUrl"` // Apisix Admin API 地址
	AdminKey string `yaml:"adminKey"` // Apisix Admin API Key
}

// Docker 配置
type DockerConfig struct {
	Host          string            `yaml:"host"`          // Docker 连接地址
	ContainerRoot string            `yaml:"containerRoot"` // 容器数据根目录
	Registries    []*DockerRegistry `yaml:"registries"`    // 镜像仓库配置列表
}

// 镜像仓库配置
type DockerRegistry struct {
	Name        string `yaml:"name"`        // 仓库名称（用于显示）
	Description string `yaml:"description"` // 仓库描述（可选）
	URL         string `yaml:"url"`         // 仓库地址，如 registry.example.com
	Username    string `yaml:"username"`    // 用户名（可选）
	Password    string `yaml:"password"`    // 密码（可选）
}

// 应用市场配置
type MarketplaceConfig struct {
	URL string `yaml:"url"` // 应用市场站点地址，通过 iframe 嵌入
}

// 工具栏链接配置
type LinkConfig struct {
	Label string `yaml:"label"` // 显示名称
	URL   string `yaml:"url"`   // 链接地址
	Icon  string `yaml:"icon"`  // Font Awesome 图标类名（可选，如 fa-link）
}

// 成员配置
type MemberConfig struct {
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HomeDirectory string `yaml:"homeDirectory"`
	// Permissions 各模块权限，key 为模块名，value 为 "r"（只读）或 "rw"（读写），空字符串或缺失表示无权限
	// 可用模块：filer, agent, apisix, docker, swarm, compose, system, shell
	Permissions map[string]string `yaml:"permissions,omitempty"`
}
