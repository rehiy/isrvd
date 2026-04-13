package config

// 配置结构
type Config struct {
	Server  *Server         `yaml:"server"`
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

// Docker 配置
type DockerConfig struct {
	Host          string `yaml:"host"`          // Docker 连接地址
	ContainerRoot string `yaml:"containerRoot"` // 容器数据根目录
}

// 成员配置
type MemberConfig struct {
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HomeDirectory string `yaml:"homeDirectory"`
	AllowTerminal bool   `yaml:"allowTerminal"`
}
