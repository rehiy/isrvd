package config

import "path/filepath"

var (
	// Server 服务器配置
	Server = ServerNormalize(nil)
	// OIDC 配置
	OIDC = OIDCNormalize(nil)
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

// Load 从配置提供者加载配置
func Load() error {
	conf, err := provider.Load()
	if err != nil {
		return err
	}

	Apply(conf)
	return nil
}

// Save 将当前全局配置保存到配置文件
func Save() error {
	members := make([]*MemberConfig, 0, len(Members))
	for _, m := range Members {
		members = append(members, m)
	}

	conf := &Config{
		Server:      Server,
		OIDC:        OIDC,
		Agent:       Agent,
		Apisix:      Apisix,
		Caddy:       Caddy,
		Docker:      Docker,
		Monitor:     Monitor,
		Marketplace: Marketplace,
		Links:       Links,
		Members:     members,
	}

	return provider.Save(conf)
}

// Apply 应用配置到全局变量（不存储）
func Apply(conf *Config) {
	if conf == nil {
		return
	}

	Server = ServerNormalize(conf.Server)

	OIDC = OIDCNormalize(conf.OIDC)

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
		Docker.ContainerRoot = PathToAbs(Server.RootDirectory, Docker.ContainerRoot)
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
		m.HomeDirectory = PathToAbs(Server.RootDirectory, m.HomeDirectory)
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
