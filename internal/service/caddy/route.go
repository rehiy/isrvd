package caddy

import (
	"context"
	"errors"

	pkgCaddy "isrvd/pkgs/caddy"
)

var ErrRouteNotFound = errors.New("路由不存在")

// ─── 路由 CRUD ───

// RouteView 路由响应视图，在 pkgCaddy.Route 基础上附加列表下标
type RouteView struct {
	Index int `json:"index"` // 路由在列表中的下标（用于定位/删除）
	pkgCaddy.Route
}

// RouteList 列出指定 server 的所有路由
func (s *Service) RouteList(ctx context.Context, server string) ([]RouteView, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return nil, err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return nil, ErrServerNotFound
	}
	out := make([]RouteView, len(srv.Routes))
	for i, r := range srv.Routes {
		out[i] = RouteView{Index: i, Route: r}
	}
	return out, nil
}

// RouteInspect 获取单条路由
func (s *Service) RouteInspect(ctx context.Context, server string, index int) (*RouteView, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return nil, err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return nil, ErrServerNotFound
	}
	if index < 0 || index >= len(srv.Routes) {
		return nil, ErrRouteNotFound
	}
	v := &RouteView{Index: index, Route: srv.Routes[index]}
	return v, nil
}

// RouteCreate 追加一条路由，返回新下标
func (s *Service) RouteCreate(ctx context.Context, server string, req pkgCaddy.Route) (int, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return -1, err
	}
	index := -1
	err = s.client.ConfigMutate(ctx, func(cfg *pkgCaddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		srv.Routes = append(srv.Routes, req)
		index = len(srv.Routes) - 1
		return nil
	})
	if err != nil {
		return -1, err
	}
	return index, nil
}

// RouteUpdate 更新指定下标的路由
func (s *Service) RouteUpdate(ctx context.Context, server string, index int, req pkgCaddy.Route) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.client.ConfigMutate(ctx, func(cfg *pkgCaddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if index < 0 || index >= len(srv.Routes) {
			return ErrRouteNotFound
		}
		srv.Routes[index] = req
		return nil
	})
}

// RouteDelete 删除指定下标的路由
func (s *Service) RouteDelete(ctx context.Context, server string, index int) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.client.ConfigMutate(ctx, func(cfg *pkgCaddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if index < 0 || index >= len(srv.Routes) {
			return ErrRouteNotFound
		}
		srv.Routes = append(srv.Routes[:index], srv.Routes[index+1:]...)
		return nil
	})
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
		form.GracePeriod = cfg.Apps.HTTP.GracePeriod.String()
	}

	return form, nil
}

// GlobalUpdate 更新全局选项
func (s *Service) GlobalUpdate(ctx context.Context, req GlobalForm) error {
	return s.client.ConfigMutate(ctx, func(cfg *pkgCaddy.Config) error {
		// srv0 是全局表单的明确目标；缺失时不隐式创建带 :80 的服务。
		srv := getServer(cfg, DefaultServerName)
		if srv == nil {
			return ErrServerNotFound
		}

		// 日志：只修改表单管理的字段，保留 writer、sampling、include 等配置。
		var log *pkgCaddy.Log
		if cfg.Logging != nil && cfg.Logging.Logs != nil {
			log = cfg.Logging.Logs["default"]
		}
		if log != nil || req.LogLevel != "" || req.LogFormat != "" {
			if cfg.Logging == nil {
				cfg.Logging = &pkgCaddy.LoggingConfig{}
			}
			if cfg.Logging.Logs == nil {
				cfg.Logging.Logs = map[string]*pkgCaddy.Log{}
			}
			if log == nil {
				log = &pkgCaddy.Log{}
				cfg.Logging.Logs["default"] = log
			}
			log.Level = req.LogLevel
			if log.Encoder == nil && req.LogFormat != "" {
				log.Encoder = map[string]any{}
			}
			if log.Encoder != nil {
				if req.LogFormat == "" {
					delete(log.Encoder, "format")
				} else {
					log.Encoder["format"] = req.LogFormat
				}
			}
		}

		issuer := buildIssuer(req)
		var auto *pkgCaddy.TLSAutomation
		if cfg.Apps != nil && cfg.Apps.TLS != nil {
			auto = cfg.Apps.TLS.Automation
		}
		if auto != nil || issuer != nil || req.OnDemandTLS {
			if cfg.Apps.TLS == nil {
				cfg.Apps.TLS = &pkgCaddy.TLSApp{}
			}
			if auto == nil {
				auto = &pkgCaddy.TLSAutomation{}
				cfg.Apps.TLS.Automation = auto
			}

			// permission 本身不会启用 on-demand；关闭策略时保留其未暴露参数。
			if req.OnDemandTLS {
				if auto.OnDemand == nil {
					auto.OnDemand = map[string]any{}
				}
				permission, _ := auto.OnDemand["permission"].(map[string]any)
				if permission == nil {
					permission = map[string]any{}
				}
				permission["module"] = "http"
				if req.OnDemandAsk == "" {
					delete(permission, "endpoint")
				} else {
					permission["endpoint"] = req.OnDemandAsk
				}
				auto.OnDemand["permission"] = permission
			}

			globalPolicyIdx := -1
			for i, policy := range auto.Policies {
				if len(policy.Subjects) == 0 {
					globalPolicyIdx = i
					break
				}
			}
			if globalPolicyIdx >= 0 {
				policy := &auto.Policies[globalPolicyIdx]
				policy.OnDemand = req.OnDemandTLS
				if issuer == nil {
					policy.Issuers = nil
				} else {
					policy.Issuers = []map[string]any{issuer}
				}
			} else if issuer != nil || req.OnDemandTLS {
				policy := pkgCaddy.TLSPolicy{OnDemand: req.OnDemandTLS}
				if issuer != nil {
					policy.Issuers = []map[string]any{issuer}
				}
				auto.Policies = append([]pkgCaddy.TLSPolicy{policy}, auto.Policies...)
			}
		}

		if srv.AutomaticHTTPS == nil && (req.AutoHTTPSDisable || req.AutoHTTPSDisableRedirects) {
			srv.AutomaticHTTPS = &pkgCaddy.AutomaticHTTPS{}
		}
		// automatic_https 是全局表单的完整可编辑字段，清除旧的显式
		// {} 存在性标记，使关闭所有开关时可恢复为未配置状态。
		delete(srv.Extras, "automatic_https")
		if srv.AutomaticHTTPS != nil {
			srv.AutomaticHTTPS.Disable = req.AutoHTTPSDisable
			srv.AutomaticHTTPS.DisableRedirects = req.AutoHTTPSDisableRedirects
			if !req.AutoHTTPSDisable && !req.AutoHTTPSDisableRedirects && !hasUnmanagedAutomaticHTTPS(srv.AutomaticHTTPS) {
				srv.AutomaticHTTPS = nil
			}
		}

		cfg.Apps.HTTP.GracePeriod = pkgCaddy.Duration(req.GracePeriod)
		return nil
	})
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
