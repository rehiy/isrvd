package caddy

import (
	"context"
	"fmt"

	pkgcaddy "isrvd/pkgs/caddy"
)

// ─── 路由 CRUD ───

// RouteView 路由响应视图，在 pkgcaddy.Route 基础上附加列表下标
type RouteView struct {
	Index int `json:"index"`
	pkgcaddy.Route
}

// RouteList 列出指定 server 的所有路由
func (s *Service) RouteList(ctx context.Context, server string) ([]RouteView, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return []RouteView{}, nil
	}
	out := make([]RouteView, len(srv.Routes))
	for i, r := range srv.Routes {
		out[i] = RouteView{Index: i, Route: r}
	}
	return out, nil
}

// RouteInspect 获取单条路由
func (s *Service) RouteInspect(ctx context.Context, server string, index int) (*RouteView, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil || index < 0 || index >= len(srv.Routes) {
		return nil, fmt.Errorf("路由不存在")
	}
	v := &RouteView{Index: index, Route: srv.Routes[index]}
	return v, nil
}

// RouteCreate 追加一条路由，返回新下标
func (s *Service) RouteCreate(ctx context.Context, server string, req pkgcaddy.Route) (int, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return -1, err
	}
	srv := ensureServer(cfg, server)
	srv.Routes = append(srv.Routes, req)
	if err := s.client.ConfigLoad(ctx, cfg); err != nil {
		return -1, err
	}
	return len(srv.Routes) - 1, nil
}

// RouteUpdate 更新指定下标的路由
func (s *Service) RouteUpdate(ctx context.Context, server string, index int, req pkgcaddy.Route) error {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || index < 0 || index >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	srv.Routes[index] = req
	return s.client.ConfigLoad(ctx, cfg)
}

// RouteDelete 删除指定下标的路由
func (s *Service) RouteDelete(ctx context.Context, server string, index int) error {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || index < 0 || index >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	srv.Routes = append(srv.Routes[:index], srv.Routes[index+1:]...)
	return s.client.ConfigLoad(ctx, cfg)
}

// ─── 全局选项 ───

// GlobalForm 全局选项编辑表单
//
// 只暴露不影响 iSrvd 管理能力的字段；
// admin 相关（listen/disabled）由 iSrvd 连接 Caddy，不允许通过此接口修改。
type GlobalForm struct {
	// 日志
	LogLevel  string `json:"logLevel,omitempty"`  // 全局日志级别：DEBUG / INFO / WARN / ERROR
	LogFormat string `json:"logFormat,omitempty"` // 日志格式：json / console，留空使用默认

	// 配置持久化与存储后端由 caddy.json 默认配置固定，不通过接口读写

	// TLS 自动化（全局默认策略，作用于无 subjects 的默认策略）
	Email      string `json:"email,omitempty"`      // ACME 注册邮箱
	AcmeCA     string `json:"acmeCA,omitempty"`     // 自定义 ACME 目录 URL，留空使用 Let's Encrypt
	LocalCerts bool   `json:"localCerts,omitempty"` // 使用本地自签证书（internal issuer），不走 ACME

	// 按需签发
	OnDemandTLS bool   `json:"onDemandTLS,omitempty"` // 启用 on_demand TLS（连接时动态申请证书）
	OnDemandAsk string `json:"onDemandAsk,omitempty"` // ask 鉴权端点 URL（防滥用，Caddy v2.8+ 必须配置）

	// automatic_https（server 级，作用于默认 server srv0）
	AutoHTTPSDisable          bool `json:"autoHttpsDisable,omitempty"`          // 禁用自动 HTTPS
	AutoHTTPSDisableRedirects bool `json:"autoHttpsDisableRedirects,omitempty"` // 禁用 HTTP→HTTPS 自动跳转

	// HTTP app 全局参数
	GracePeriod string `json:"gracePeriod,omitempty"` // 优雅关闭等待时间，例如 10s（apps.http.grace_period）
}

// Global 获取全局选项
func (s *Service) Global(ctx context.Context) (*GlobalForm, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	form := &GlobalForm{}

	// 日志
	if cfg.Logging != nil {
		if log, ok := cfg.Logging.Logs["default"]; ok && log != nil {
			form.LogLevel = log.Level
			if enc := log.Encoder; enc != nil {
				form.LogFormat, _ = enc["format"].(string)
			}
		}
	}

	// 配置持久化与存储后端由 caddy.json 默认配置固定，不通过接口读写

	// TLS 自动化：从全局默认策略（无 subjects）读取
	if cfg.Apps != nil && cfg.Apps.TLS != nil && cfg.Apps.TLS.Automation != nil {
		auto := cfg.Apps.TLS.Automation

		// on_demand permission（ask 端点）
		if auto.OnDemand != nil {
			if perm, ok := auto.OnDemand["permission"].(map[string]any); ok {
				form.OnDemandAsk, _ = perm["endpoint"].(string)
			}
		}

		// 全局默认策略：第一个无 subjects 的策略
		for _, p := range auto.Policies {
			if len(p.Subjects) > 0 {
				continue
			}
			// on_demand 以策略级开关为准（全局 permission + 策略 on_demand 两者均需启用）
			form.OnDemandTLS = p.OnDemand
			for _, issuer := range p.Issuers {
				mod, _ := issuer["module"].(string)
				switch mod {
				case "internal":
					form.LocalCerts = true
				case "acme", "zerossl":
					if v, ok := issuer["email"].(string); ok {
						form.Email = v
					}
					if v, ok := issuer["ca"].(string); ok {
						form.AcmeCA = v
					}
				}
			}
			break
		}
	}

	// automatic_https（server 级，作用于默认 server）
	if cfg.Apps != nil && cfg.Apps.HTTP != nil {
		if srv, ok := cfg.Apps.HTTP.Servers[DefaultServerName]; ok && srv != nil && srv.AutomaticHTTPS != nil {
			form.AutoHTTPSDisable = srv.AutomaticHTTPS.Disable
			form.AutoHTTPSDisableRedirects = srv.AutomaticHTTPS.DisableRedirects
		}
	}

	// HTTP app 全局参数
	if cfg.Apps != nil && cfg.Apps.HTTP != nil {
		form.GracePeriod = cfg.Apps.HTTP.GracePeriod
	}

	return form, nil
}

// GlobalUpdate 更新全局选项
func (s *Service) GlobalUpdate(ctx context.Context, req GlobalForm) error {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}

	// 日志
	if req.LogLevel != "" || req.LogFormat != "" {
		if cfg.Logging == nil {
			cfg.Logging = &pkgcaddy.LoggingConfig{}
		}
		if cfg.Logging.Logs == nil {
			cfg.Logging.Logs = map[string]*pkgcaddy.Log{}
		}
		log := &pkgcaddy.Log{}
		if req.LogLevel != "" {
			log.Level = req.LogLevel
		}
		if req.LogFormat != "" {
			log.Encoder = map[string]any{"format": req.LogFormat}
		}
		cfg.Logging.Logs["default"] = log
	} else if cfg.Logging != nil {
		delete(cfg.Logging.Logs, "default")
	}

	// 配置持久化与存储后端由 caddy.json 默认配置固定，此处不修改

	// TLS 自动化
	if cfg.Apps == nil {
		cfg.Apps = &pkgcaddy.AppsConfig{}
	}
	if cfg.Apps.TLS == nil {
		cfg.Apps.TLS = &pkgcaddy.TLSApp{}
	}
	if cfg.Apps.TLS.Automation == nil {
		cfg.Apps.TLS.Automation = &pkgcaddy.TLSAutomation{}
	}
	auto := cfg.Apps.TLS.Automation

	// on_demand：需同时设置全局 permission 和默认策略的 on_demand: true
	if req.OnDemandTLS {
		perm := map[string]any{"module": "http"}
		if req.OnDemandAsk != "" {
			perm["endpoint"] = req.OnDemandAsk
		}
		auto.OnDemand = map[string]any{"permission": perm}
	} else {
		auto.OnDemand = nil
	}

	// 全局默认策略（无 subjects），重新构建 issuers 和 on_demand
	globalPolicyIdx := -1
	for i, p := range auto.Policies {
		if len(p.Subjects) == 0 {
			globalPolicyIdx = i
			break
		}
	}
	issuer := buildIssuer(req)
	if issuer != nil || req.OnDemandTLS {
		policy := pkgcaddy.TLSPolicy{
			Issuers:  []map[string]any{},
			OnDemand: req.OnDemandTLS,
		}
		if issuer != nil {
			policy.Issuers = []map[string]any{issuer}
		} else {
			policy.Issuers = nil
		}
		if globalPolicyIdx >= 0 {
			auto.Policies[globalPolicyIdx] = policy
		} else {
			auto.Policies = append([]pkgcaddy.TLSPolicy{policy}, auto.Policies...)
		}
	} else if globalPolicyIdx >= 0 {
		auto.Policies = append(auto.Policies[:globalPolicyIdx], auto.Policies[globalPolicyIdx+1:]...)
	}

	// 自动化为空时清理
	if len(auto.Policies) == 0 && auto.OnDemand == nil {
		cfg.Apps.TLS.Automation = nil
	}

	// automatic_https（server 级，作用于默认 server srv0）
	srv := ensureServer(cfg, DefaultServerName)
	if req.AutoHTTPSDisable || req.AutoHTTPSDisableRedirects {
		srv.AutomaticHTTPS = &pkgcaddy.AutomaticHTTPS{
			Disable:          req.AutoHTTPSDisable,
			DisableRedirects: req.AutoHTTPSDisableRedirects,
		}
	} else {
		srv.AutomaticHTTPS = nil
	}

	// HTTP app 优雅关闭
	cfg.Apps.HTTP.GracePeriod = req.GracePeriod

	return s.client.ConfigLoad(ctx, cfg)
}

// buildIssuer 根据表单构造 issuer map；无有效配置时返回 nil
func buildIssuer(req GlobalForm) map[string]any {
	if req.LocalCerts {
		return map[string]any{"module": "internal"}
	}
	if req.Email != "" || req.AcmeCA != "" {
		m := map[string]any{"module": "acme"}
		if req.Email != "" {
			m["email"] = req.Email
		}
		if req.AcmeCA != "" {
			m["ca"] = req.AcmeCA
		}
		return m
	}
	return nil
}
