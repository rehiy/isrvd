package config

// 配置结构
type Config struct {
	Server  *Server   `yaml:"server"`
	Members []*Member `yaml:"members"`
}

// 服务器配置
type Server struct {
	Debug         bool   `yaml:"debug"`
	ListenAddr    string `yaml:"listenAddr"`
	RootDirectory string `yaml:"baseDirectory"`
	JWTSecret     string `yaml:"jwtSecret"`
}

// 成员配置
type Member struct {
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	HomeDirectory string `yaml:"homeDirectory"`
	AllowTerminal bool   `yaml:"allowTerminal"`
}
