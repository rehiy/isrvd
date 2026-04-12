package config

// 配置结构
type Config struct {
	Server  *Server   `yaml:"server"`
	Members []*Member `yaml:"members"`
}

// 服务器配置
type Server struct {
	Debug           bool   `yaml:"debug"`
	ListenAddr      string `yaml:"listenAddr"`
	JWTSecret       string `yaml:"jwtSecret"`
	ProxyHeaderName string `yaml:"proxyHeaderName"`
	RootDirectory   string `yaml:"rootDirectory"`
	ContainerRoot   string `yaml:"containerRoot"` // 容器数据根目录
}

// 成员配置
type Member struct {
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HomeDirectory string `yaml:"homeDirectory"`
	AllowTerminal bool   `yaml:"allowTerminal"`
}
