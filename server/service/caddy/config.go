package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"isrvd/pkgs/caddy"
)

// ─── 概览与原始配置 ───

// Info Caddy 概览信息
type Info struct {
	Servers   int  `json:"servers"`   // HTTP server 数量
	Routes    int  `json:"routes"`    // 路由总数
	Certs     int  `json:"certs"`     // 证书数量
	HasTLS    bool `json:"hasTls"`    // 是否配置了 TLS
	Available bool `json:"available"` // Caddy 服务是否可用
}

// Info 获取概览
func (s *Service) Info(ctx context.Context) (*Info, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return &Info{Available: false}, nil
	}
	info := &Info{Available: true}
	if cfg.Apps != nil && cfg.Apps.HTTP != nil {
		for _, srv := range cfg.Apps.HTTP.Servers {
			info.Servers++
			info.Routes += len(srv.Routes)
		}
	}
	if cfg.Apps != nil && cfg.Apps.TLS != nil {
		info.HasTLS = true
	}
	info.Certs = len(s.certListFromConfig(ctx, cfg))
	return info, nil
}

// ConfigAll 获取完整配置（原始 JSON）
func (s *Service) ConfigAll(ctx context.Context) (json.RawMessage, error) {
	raw, err := s.client.ConfigRaw(ctx, "")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(raw), nil
}

// ConfigLoad 整体替换配置
func (s *Service) ConfigLoad(ctx context.Context, raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("配置内容不能为空")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return fmt.Errorf("config 必须是 JSON 对象")
	}
	return s.client.ConfigLoadRaw(ctx, raw)
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
			_, issuer := formIssuer(p.Issuers)
			form.LocalCerts = issuer.LocalCerts
			form.Email, form.AcmeCA = issuer.Email, issuer.AcmeCA
			break
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
	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		if cfg.Apps == nil {
			cfg.Apps = &caddy.AppsConfig{}
		}
		if cfg.Apps.HTTP == nil {
			cfg.Apps.HTTP = &caddy.HTTPApp{}
		}

		// 日志：只修改表单管理的字段，保留 writer、sampling、include 等配置。
		var log *caddy.Log
		if cfg.Logging != nil && cfg.Logging.Logs != nil {
			log = cfg.Logging.Logs["default"]
		}
		if log != nil || req.LogLevel != "" || req.LogFormat != "" {
			if cfg.Logging == nil {
				cfg.Logging = &caddy.LoggingConfig{}
			}
			if cfg.Logging.Logs == nil {
				cfg.Logging.Logs = map[string]*caddy.Log{}
			}
			if log == nil {
				log = &caddy.Log{}
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
		var auto *caddy.TLSAutomation
		if cfg.Apps != nil && cfg.Apps.TLS != nil {
			auto = cfg.Apps.TLS.Automation
		}
		if auto != nil || issuer != nil || req.OnDemandTLS {
			if cfg.Apps.TLS == nil {
				cfg.Apps.TLS = &caddy.TLSApp{}
			}
			if auto == nil {
				auto = &caddy.TLSAutomation{}
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

			globalPolicyIdx := slices.IndexFunc(auto.Policies, func(policy caddy.TLSPolicy) bool {
				return len(policy.Subjects) == 0
			})
			if globalPolicyIdx >= 0 {
				policy := &auto.Policies[globalPolicyIdx]
				policy.OnDemand = req.OnDemandTLS
				policy.Issuers = updateIssuer(policy.Issuers, req, issuer)
			} else if issuer != nil || req.OnDemandTLS {
				policy := caddy.TLSPolicy{OnDemand: req.OnDemandTLS}
				if issuer != nil {
					policy.Issuers = []map[string]any{issuer}
				}
				auto.Policies = append([]caddy.TLSPolicy{policy}, auto.Policies...)
			}
		}

		cfg.Apps.HTTP.GracePeriod = caddy.Duration(req.GracePeriod)
		return nil
	})
}

// ─── 辅助函数 ───

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

// formIssuer 只投影一个受表单管理的签发器，其余配置留在原策略中。
func formIssuer(issuers []map[string]any) (int, GlobalForm) {
	for index, issuer := range issuers {
		switch issuer["module"] {
		case "internal":
			return index, GlobalForm{LocalCerts: true}
		case "acme", "zerossl":
			email, _ := issuer["email"].(string)
			ca, _ := issuer["ca"].(string)
			return index, GlobalForm{Email: email, AcmeCA: ca}
		}
	}
	return -1, GlobalForm{}
}

func updateIssuer(issuers []map[string]any, req GlobalForm, desired map[string]any) []map[string]any {
	index, previous := formIssuer(issuers)
	if previous.LocalCerts == req.LocalCerts && previous.Email == req.Email && previous.AcmeCA == req.AcmeCA {
		return issuers
	}
	if index < 0 {
		if desired != nil {
			return append(issuers, desired)
		}
		return issuers
	}
	current := issuers[index]
	if previous.LocalCerts != req.LocalCerts {
		if desired == nil {
			return slices.Delete(issuers, index, index+1)
		}
		issuers[index] = desired
		return issuers
	}
	if !req.LocalCerts {
		for _, field := range []string{"email", "ca"} {
			if value, ok := desired[field]; ok {
				current[field] = value
			} else {
				delete(current, field)
			}
		}
	}
	return issuers
}
