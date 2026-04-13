package config

// 配置结构
type Config struct {
	Server  *Server         `yaml:"server"`
	Apisix  *ApisixConfig   `yaml:"apisix"`
	Docker  *DockerConfig   `yaml:"docker"`
	Members []*MemberConfig `yaml:"members"`
}

// 服务器配置
type Server struct {
	Debug           bool   `yaml:"debug"`
	ListenAddr      string `yaml:"listenAddr"`
	JWTSecret       string `yaml:"jwtSecret"`
	ProxyHeaderName string `yaml:"proxyHeaderName"`
	RootDirectory   string `yaml:"rootDirectory"`
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
	Name     string `yaml:"name"`     // 仓库名称（用于显示）
	URL      string `yaml:"url"`      // 仓库地址，如 registry.example.com
	Username string `yaml:"username"` // 用户名（可选）
	Password string `yaml:"password"` // 密码（可选）
}

// 成员配置
type MemberConfig struct {
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HomeDirectory string `yaml:"homeDirectory"`
	AllowTerminal bool   `yaml:"allowTerminal"`
}
