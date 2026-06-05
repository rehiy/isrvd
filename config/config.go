package config

import "path/filepath"

var (
	// Server 服务器配置
	Server = ServerNormalize(nil)
	// THA 可信头认证配置
	THA = &THAConfig{}
	// OIDC 配置
	OIDC = OIDCNormalize(nil)
	// Passkey 配置
	Passkey = &PasskeyConfig{}
	// Agent LLM 配置
	Agent = &AgentConfig{}
	// Apisix 配置
	Apisix = &ApisixConfig{}
	// Caddy 配置
	Caddy = &CaddyConfig{}
	// Docker 配置
	Docker = &DockerConfig{}
	// Monitor 监控配置
	Monitor = MonitorNormalize(nil)
	// 应用市场配置
	Marketplace = &MarketplaceConfig{}
	// 工具栏链接配置
	Links []*LinkConfig
	// 成员配置
	Members = map[string]*MemberConfig{}
	// 版本信息（编译时通过脚本注入）
	Version = "v0.0.0"
)

// Apply 应用配置到全局变量（不存储）
func Apply(conf *Config) {
	if conf == nil {
		return
	}

	Server = ServerNormalize(conf.Server)

	if conf.THA != nil {
		THA = THANormalize(conf.THA)
	}

	OIDC = OIDCNormalize(conf.OIDC)

	Passkey = PasskeyNormalize(conf.Passkey)

	if conf.Agent != nil {
		Agent = conf.Agent
	}

	if conf.Apisix != nil {
		Apisix = conf.Apisix
	}

	if conf.Caddy != nil {
		Caddy = conf.Caddy
	}

	if conf.Docker != nil {
		Docker = conf.Docker
		Docker.ContainerRoot = PathToAbs(Docker.ContainerRoot, Server.RootDirectory)
	}

	Monitor = MonitorNormalize(conf.Monitor)

	if conf.Marketplace != nil {
		Marketplace = conf.Marketplace
	}

	if conf.Links != nil {
		Links = conf.Links
	}

	Members = make(map[string]*MemberConfig, len(conf.Members))
	for _, m := range conf.Members {
		m.HomeDirectory = PathToAbs(m.HomeDirectory, Server.RootDirectory)
		Members[m.Username] = m
	}
}

// ServerNormalize 填充 Server 默认值并归一化路径
func ServerNormalize(server *ServerConfig) *ServerConfig {
	if server == nil {
		server = &ServerConfig{}
	}
	if server.ListenAddr == "" {
		server.ListenAddr = ":8080"
	}
	if server.JWTExpiration == 0 {
		server.JWTExpiration = 86400
	}
	if server.MaxUploadSize == 0 {
		server.MaxUploadSize = 100 << 20
	}
	if server.RootDirectory == "" {
		server.RootDirectory = "."
	}
	if !filepath.IsAbs(server.RootDirectory) {
		if abs, err := filepath.Abs(server.RootDirectory); err == nil {
			server.RootDirectory = abs
		}
	}
	return server
}

// MonitorNormalize 填充 Monitor 默认值
func MonitorNormalize(monitor *MonitorConfig) *MonitorConfig {
	if monitor == nil {
		monitor = &MonitorConfig{}
	}
	// Interval 合法值：5、15、30、60；其他值均视为禁用，置 0
	switch monitor.Interval {
	case 5, 15, 30, 60:
		// 合法值，保留
	default:
		monitor.Interval = 0
	}
	return monitor
}

// OIDCNormalize 填充 OIDC 默认值
func OIDCNormalize(oidc *OIDCConfig) *OIDCConfig {
	if oidc == nil {
		oidc = &OIDCConfig{}
	}
	if oidc.UsernameClaim == "" {
		oidc.UsernameClaim = "sub"
	}
	if len(oidc.Scopes) == 0 {
		oidc.Scopes = []string{"openid", "profile", "email"}
	}
	return oidc
}

// PasskeyNormalize 填充 Passkey 默认值
func PasskeyNormalize(passkey *PasskeyConfig) *PasskeyConfig {
	if passkey == nil {
		passkey = &PasskeyConfig{}
	}
	if passkey.Timeout == 0 {
		passkey.Timeout = 60000
	}
	return passkey
}

// THANormalize 填充 THA 默认值
func THANormalize(tha *THAConfig) *THAConfig {
	if tha == nil {
		tha = &THAConfig{}
	}
	if tha.HeaderName == "" {
		tha.HeaderName = "X-Username"
	}
	return tha
}
