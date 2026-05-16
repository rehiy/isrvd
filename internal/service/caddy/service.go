// Package caddy 提供 Caddy Admin API 业务服务层
//
// 资源映射（业务约定）：
//   - 单 server 模式：默认 server 名为 srv0
//   - Route：servers/srv0/routes 数组项，对外用数组下标做主键
//   - 编辑器使用 RouteForm 简化模型，service 层负责与 Caddy 原生 Handler/Match 互转
package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	"isrvd/internal/registry"
	pkgcaddy "isrvd/pkgs/caddy"
)

// DefaultServerName 默认 server 名（业务约定，唯一一份）
const DefaultServerName = "srv0"

// HandlerKindReverseProxy 等：路由后端类型
const (
	HandlerKindReverseProxy = "reverse_proxy"
	HandlerKindFileServer   = "file_server"
	HandlerKindStaticResp   = "static_response"
	HandlerKindRaw          = "raw" // 透传原始 handle 数组
)

// CertSource 证书来源类型
const (
	CertSourceFile     = "file"     // load_files：磁盘文件路径
	CertSourcePEM      = "pem"      // load_pem：内联 PEM
	CertSourceAutomate = "automate" // automation.policies[].subjects：自动签发
)

// Service Caddy 业务服务
type Service struct {
	client *pkgcaddy.Client
}

// NewService 创建 Caddy 业务服务
func NewService() (*Service, error) {
	client := registry.CaddyClient
	if client == nil {
		logman.Error("Caddy client not initialized")
		return nil, fmt.Errorf("Caddy 未配置")
	}
	return &Service{client: client}, nil
}

// ─── 概览与原始配置 ───

// Info Caddy 概览信息
type Info struct {
	AdminURL  string `json:"adminUrl"`
	Servers   int    `json:"servers"`
	Routes    int    `json:"routes"`
	HasTLS    bool   `json:"hasTls"`
	Available bool   `json:"available"`
}

// Info 获取概览
func (s *Service) Info(ctx context.Context) (*Info, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return &Info{AdminURL: config.Caddy.AdminURL, Available: false}, nil
	}
	info := &Info{AdminURL: config.Caddy.AdminURL, Available: true}
	if cfg.Apps != nil && cfg.Apps.HTTP != nil {
		for _, srv := range cfg.Apps.HTTP.Servers {
			info.Servers++
			info.Routes += len(srv.Routes)
		}
	}
	if cfg.Apps != nil && cfg.Apps.TLS != nil {
		info.HasTLS = true
	}
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

// ─── 路由 CRUD ───

// MatchForm 简化的 match 编辑模型
type MatchForm struct {
	Hosts   []string `json:"hosts,omitempty"`
	Paths   []string `json:"paths,omitempty"`
	Methods []string `json:"methods,omitempty"`
}

// HandlerForm 简化的 handler 编辑模型，按 Kind 解释字段
type HandlerForm struct {
	Kind string `json:"kind"` // reverse_proxy / file_server / static_response / raw

	// reverse_proxy
	Upstreams []string `json:"upstreams,omitempty"`

	// file_server
	Root   string `json:"root,omitempty"`
	Browse bool   `json:"browse,omitempty"`

	// static_response
	StatusCode int    `json:"statusCode,omitempty"`
	Body       string `json:"body,omitempty"`

	// raw：透传原始 handle 数组
	Raw json.RawMessage `json:"raw,omitempty"`
}

// RouteForm 路由编辑表单
type RouteForm struct {
	Index    int          `json:"index"` // 数组下标，仅响应使用
	Group    string       `json:"group,omitempty"`
	Match    *MatchForm   `json:"match,omitempty"`
	Handler  *HandlerForm `json:"handler,omitempty"`
	Terminal bool         `json:"terminal,omitempty"`
	ID       string       `json:"id,omitempty"`
}

// RouteList 列出指定 server 的所有路由
//
// server 为空时使用 DefaultServerName。
func (s *Service) RouteList(ctx context.Context, server string) ([]RouteForm, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return []RouteForm{}, nil
	}
	out := make([]RouteForm, 0, len(srv.Routes))
	for i, r := range srv.Routes {
		out = append(out, routeToForm(i, r))
	}
	return out, nil
}

// RouteInspect 获取单条路由
func (s *Service) RouteInspect(ctx context.Context, server string, index int) (*RouteForm, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil || index < 0 || index >= len(srv.Routes) {
		return nil, fmt.Errorf("路由不存在")
	}
	form := routeToForm(index, srv.Routes[index])
	return &form, nil
}

// RouteCreate 追加一条路由，返回新下标
func (s *Service) RouteCreate(ctx context.Context, server string, req RouteForm) (int, error) {
	if err := validateRouteForm(req); err != nil {
		return -1, err
	}
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return -1, err
	}
	srv := ensureServer(cfg, server)
	route, err := formToRoute(req)
	if err != nil {
		return -1, err
	}
	srv.Routes = append(srv.Routes, route)
	if err := s.client.ConfigLoad(ctx, cfg); err != nil {
		return -1, err
	}
	return len(srv.Routes) - 1, nil
}

// RouteUpdate 更新指定下标的路由
func (s *Service) RouteUpdate(ctx context.Context, server string, index int, req RouteForm) error {
	if err := validateRouteForm(req); err != nil {
		return err
	}
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || index < 0 || index >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	route, err := formToRoute(req)
	if err != nil {
		return err
	}
	srv.Routes[index] = route
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

// ─── 辅助：表单 ↔ Caddy 原生模型 ───

func validateRouteForm(req RouteForm) error {
	if req.Handler == nil || req.Handler.Kind == "" {
		return fmt.Errorf("handler 不能为空")
	}
	switch req.Handler.Kind {
	case HandlerKindReverseProxy:
		ups := nonEmpty(req.Handler.Upstreams)
		if len(ups) == 0 {
			return fmt.Errorf("反向代理上游不能为空")
		}
	case HandlerKindFileServer:
		if strings.TrimSpace(req.Handler.Root) == "" {
			return fmt.Errorf("文件根目录不能为空")
		}
	case HandlerKindStaticResp:
		if req.Handler.StatusCode == 0 && req.Handler.Body == "" {
			return fmt.Errorf("静态响应至少需要 status 或 body")
		}
	case HandlerKindRaw:
		if len(req.Handler.Raw) == 0 {
			return fmt.Errorf("原始 handle 不能为空")
		}
	default:
		return fmt.Errorf("不支持的 handler 类型: %s", req.Handler.Kind)
	}
	return nil
}

// formToRoute 把表单转成 caddy.Route
func formToRoute(req RouteForm) (pkgcaddy.Route, error) {
	r := pkgcaddy.Route{
		Group:    req.Group,
		Terminal: req.Terminal,
		ID:       req.ID,
	}
	if req.Match != nil {
		match := pkgcaddy.MatchSet{}
		if hosts := nonEmpty(req.Match.Hosts); len(hosts) > 0 {
			match["host"] = hosts
		}
		if paths := nonEmpty(req.Match.Paths); len(paths) > 0 {
			match["path"] = paths
		}
		if methods := nonEmpty(req.Match.Methods); len(methods) > 0 {
			match["method"] = methods
		}
		if len(match) > 0 {
			r.Match = []pkgcaddy.MatchSet{match}
		}
	}

	switch req.Handler.Kind {
	case HandlerKindReverseProxy:
		r.Handle = []pkgcaddy.Handler{pkgcaddy.HandlerReverseProxy(nonEmpty(req.Handler.Upstreams)...)}
	case HandlerKindFileServer:
		r.Handle = []pkgcaddy.Handler{pkgcaddy.HandlerFileServer(req.Handler.Root, req.Handler.Browse)}
	case HandlerKindStaticResp:
		r.Handle = []pkgcaddy.Handler{pkgcaddy.HandlerStaticResponse(req.Handler.StatusCode, req.Handler.Body)}
	case HandlerKindRaw:
		var handlers []pkgcaddy.Handler
		if err := json.Unmarshal(req.Handler.Raw, &handlers); err != nil {
			return r, fmt.Errorf("原始 handle 解析失败: %w", err)
		}
		r.Handle = handlers
	}
	return r, nil
}

// routeToForm 把 caddy.Route 转成表单
func routeToForm(index int, r pkgcaddy.Route) RouteForm {
	form := RouteForm{
		Index:    index,
		Group:    r.Group,
		Terminal: r.Terminal,
		ID:       r.ID,
	}
	// match：取第一个 set
	if len(r.Match) > 0 {
		m := r.Match[0]
		mf := &MatchForm{
			Hosts:   toStrSlice(m["host"]),
			Paths:   toStrSlice(m["path"]),
			Methods: toStrSlice(m["method"]),
		}
		form.Match = mf
	}
	// handler：识别第一个 handler 的 kind；多 handler 或不识别都用 raw
	form.Handler = handlerToForm(r.Handle)
	return form
}

// handlerToForm 识别已知 handler 类型并填充字段
func handlerToForm(handlers []pkgcaddy.Handler) *HandlerForm {
	if len(handlers) == 0 {
		return &HandlerForm{Kind: HandlerKindStaticResp, StatusCode: 200}
	}
	if len(handlers) > 1 {
		return rawHandler(handlers)
	}

	h := handlers[0]
	kind, _ := h["handler"].(string)
	switch kind {
	case HandlerKindReverseProxy:
		return &HandlerForm{Kind: kind, Upstreams: extractUpstreams(h)}
	case HandlerKindFileServer:
		root, _ := h["root"].(string)
		_, browse := h["browse"]
		return &HandlerForm{Kind: kind, Root: root, Browse: browse}
	case HandlerKindStaticResp:
		body, _ := h["body"].(string)
		status := 0
		switch v := h["status_code"].(type) {
		case float64:
			status = int(v)
		case int:
			status = v
		case string:
			status, _ = strconv.Atoi(v)
		}
		return &HandlerForm{Kind: kind, StatusCode: status, Body: body}
	default:
		return rawHandler(handlers)
	}
}

func rawHandler(handlers []pkgcaddy.Handler) *HandlerForm {
	raw, _ := json.Marshal(handlers)
	return &HandlerForm{Kind: HandlerKindRaw, Raw: raw}
}

// extractUpstreams 从 reverse_proxy handler 抽取 dial 列表
func extractUpstreams(h pkgcaddy.Handler) []string {
	arr, _ := h["upstreams"].([]any)
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			if dial, _ := m["dial"].(string); dial != "" {
				out = append(out, dial)
			}
		}
	}
	return out
}

// ─── 全局选项 ───

// GlobalForm 全局选项编辑表单
//
// 只暴露不影响 isrvd 管理能力的字段；
// admin 相关（listen/disabled）由 isrvd 连接 Caddy，不允许通过此接口修改。
type GlobalForm struct {
	// 日志
	LogLevel  string `json:"logLevel,omitempty"`  // 全局日志级别：DEBUG / INFO / WARN / ERROR
	LogFormat string `json:"logFormat,omitempty"` // 日志格式：json / console，留空使用默认

	// 存储后端（顶层 storage）
	// 注意：caddy 镜像已在默认配置中固定 storage.root=/data/caddy，不建议通过此接口修改

	// TLS 自动化（全局默认策略，作用于无 subjects 的默认策略）
	Email    string `json:"email,omitempty"`    // ACME 注册邮箱
	AcmeCA   string `json:"acmeCA,omitempty"`   // 自定义 ACME 目录 URL，留空使用 Let's Encrypt
	LocalCerts bool `json:"localCerts,omitempty"` // 使用本地自签证书（internal issuer），不走 ACME

	// 按需签发
	OnDemandTLS bool `json:"onDemandTLS,omitempty"` // 启用 on_demand TLS（连接时动态申请证书）

	// automatic_https（server 级，作用于默认 server srv0）
	AutoHTTPSDisable          bool `json:"autoHttpsDisable,omitempty"`          // 禁用自动 HTTPS
	AutoHTTPSDisableRedirects bool `json:"autoHttpsDisableRedirects,omitempty"` // 禁用 HTTP→HTTPS 自动跳转

	// HTTP app 全局参数
	GracePeriod string `json:"gracePeriod,omitempty"` // 优雅关闭等待时间，例如 10s（apps.http.grace_period）
}

// GlobalGet 获取全局选项
func (s *Service) GlobalGet(ctx context.Context) (*GlobalForm, error) {
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

		// on_demand
		form.OnDemandTLS = auto.OnDemand != nil

		// 全局默认策略：第一个无 subjects 的策略
		for _, p := range auto.Policies {
			if len(p.Subjects) > 0 {
				continue
			}
			// 读取 issuers
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

	// on_demand
	if req.OnDemandTLS {
		if auto.OnDemand == nil {
			auto.OnDemand = map[string]any{}
		}
	} else {
		auto.OnDemand = nil
	}

	// 全局默认策略（无 subjects），重新构建 issuers
	globalPolicyIdx := -1
	for i, p := range auto.Policies {
		if len(p.Subjects) == 0 {
			globalPolicyIdx = i
			break
		}
	}
	issuer := buildIssuer(req)
	if issuer != nil {
		policy := pkgcaddy.TLSPolicy{Issuers: []map[string]any{issuer}}
		if globalPolicyIdx >= 0 {
			auto.Policies[globalPolicyIdx] = policy
		} else {
			// 默认策略插到最前
			auto.Policies = append([]pkgcaddy.TLSPolicy{policy}, auto.Policies...)
		}
	} else if globalPolicyIdx >= 0 {
		// 清除旧的全局默认策略
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

// ─── 路由 CRUD ───

func normalizeServer(name string) string {
	if name == "" {
		return DefaultServerName
	}
	return name
}

// getServer 取 server，不存在返回 nil
func getServer(cfg *pkgcaddy.Config, name string) *pkgcaddy.HTTPServer {
	if cfg == nil || cfg.Apps == nil || cfg.Apps.HTTP == nil {
		return nil
	}
	return cfg.Apps.HTTP.Servers[name]
}

// ensureServer 取 server，不存在则创建，并初始化 listen。
func ensureServer(cfg *pkgcaddy.Config, name string) *pkgcaddy.HTTPServer {
	if cfg.Apps == nil {
		cfg.Apps = &pkgcaddy.AppsConfig{}
	}
	if cfg.Apps.HTTP == nil {
		cfg.Apps.HTTP = &pkgcaddy.HTTPApp{}
	}
	if cfg.Apps.HTTP.Servers == nil {
		cfg.Apps.HTTP.Servers = map[string]*pkgcaddy.HTTPServer{}
	}
	srv, ok := cfg.Apps.HTTP.Servers[name]
	if !ok {
		srv = &pkgcaddy.HTTPServer{Listen: []string{":80"}}
		cfg.Apps.HTTP.Servers[name] = srv
	}
	return srv
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if v := strings.TrimSpace(s); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func toStrSlice(v any) []string {
	switch arr := v.(type) {
	case []string:
		return arr
	case []any:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ─── TLS 证书 CRUD ───

// CertForm TLS 证书统一编辑模型
//
// 通过 Source 区分三种来源：
//   - file：load_files，Certificate/Key 是文件路径
//   - pem：load_pem，Certificate/Key 是 PEM 文本
//   - automate：automation.policies[].subjects 中的 host
type CertForm struct {
	Key         string   `json:"key,omitempty"`         // 复合主键 <source>-<index>，仅响应使用
	Source      string   `json:"source"`                // file / pem / automate
	Certificate string   `json:"certificate,omitempty"` // file: 路径；pem: PEM 文本
	KeyContent  string   `json:"keyContent,omitempty"`  // 私钥（路径或 PEM 文本）
	Tags        []string `json:"tags,omitempty"`
	Format      string   `json:"format,omitempty"`  // 仅 file 类型使用
	Subject     string   `json:"subject,omitempty"` // 仅 automate 使用：host 名称
}

// CertList 列出所有证书（合并三种来源）
func (s *Service) CertList(ctx context.Context) ([]CertForm, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CertForm, 0)
	if cfg.Apps == nil || cfg.Apps.TLS == nil {
		return out, nil
	}
	tls := cfg.Apps.TLS
	if tls.Certificates != nil {
		for i, f := range tls.Certificates.LoadFiles {
			out = append(out, CertForm{
				Key:         buildCertKey(CertSourceFile, i),
				Source:      CertSourceFile,
				Certificate: f.Certificate,
				// KeyContent 不返回：file 来源是路径引用，路径本身是安全信息
				Tags:   f.Tags,
				Format: f.Format,
			})
		}
		for i, p := range tls.Certificates.LoadPEM {
			out = append(out, CertForm{
				Key:         buildCertKey(CertSourcePEM, i),
				Source:      CertSourcePEM,
				Certificate: p.Certificate,
				// KeyContent 不返回：私钥不在列表接口暴露；编辑时客户端留空=保留原值
				Tags: p.Tags,
			})
		}
	}
	if tls.Automation != nil {
		idx := 0
		for _, policy := range tls.Automation.Policies {
			for _, subject := range policy.Subjects {
				out = append(out, CertForm{
					Key:     buildCertKey(CertSourceAutomate, idx),
					Source:  CertSourceAutomate,
					Subject: subject,
				})
				idx++
			}
		}
	}
	return out, nil
}

// CertCreate 创建证书
func (s *Service) CertCreate(ctx context.Context, req CertForm) error {
	if err := validateCertForm(req, true); err != nil {
		return err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch req.Source {
	case CertSourceFile:
		ensureCerts(tls).LoadFiles = append(tls.Certificates.LoadFiles, pkgcaddy.TLSLoadFile{
			Certificate: req.Certificate,
			Key:         req.KeyContent,
			Tags:        req.Tags,
			Format:      req.Format,
		})
	case CertSourcePEM:
		ensureCerts(tls).LoadPEM = append(tls.Certificates.LoadPEM, pkgcaddy.TLSLoadPEM{
			Certificate: req.Certificate,
			Key:         req.KeyContent,
			Tags:        req.Tags,
		})
	case CertSourceAutomate:
		appendAutomateSubject(tls, req.Subject)
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// CertUpdate 更新证书（按 key 定位）
func (s *Service) CertUpdate(ctx context.Context, key string, req CertForm) error {
	source, index, err := parseCertKey(key)
	if err != nil {
		return err
	}
	if req.Source != "" && req.Source != source {
		return fmt.Errorf("不支持跨来源更新（%s → %s）", source, req.Source)
	}
	req.Source = source
	if err := validateCertForm(req, false); err != nil {
		return err
	}

	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch source {
	case CertSourceFile:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadFiles) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadFiles[index] = pkgcaddy.TLSLoadFile{
			Certificate: req.Certificate,
			// 私钥留空则保留原值（客户端编辑时可不回填路径）
			Key:    pickSecretStr(req.KeyContent, tls.Certificates.LoadFiles[index].Key),
			Tags:   req.Tags,
			Format: req.Format,
		}
	case CertSourcePEM:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadPEM) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadPEM[index] = pkgcaddy.TLSLoadPEM{
			Certificate: req.Certificate,
			// 私钥留空则保留原值
			Key:  pickSecretStr(req.KeyContent, tls.Certificates.LoadPEM[index].Key),
			Tags: req.Tags,
		}
	case CertSourceAutomate:
		if !replaceAutomateSubject(tls, index, req.Subject) {
			return fmt.Errorf("证书不存在")
		}
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// CertDelete 删除证书
func (s *Service) CertDelete(ctx context.Context, key string) error {
	source, index, err := parseCertKey(key)
	if err != nil {
		return err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch source {
	case CertSourceFile:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadFiles) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadFiles = append(tls.Certificates.LoadFiles[:index], tls.Certificates.LoadFiles[index+1:]...)
	case CertSourcePEM:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadPEM) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadPEM = append(tls.Certificates.LoadPEM[:index], tls.Certificates.LoadPEM[index+1:]...)
	case CertSourceAutomate:
		if !removeAutomateSubject(tls, index) {
			return fmt.Errorf("证书不存在")
		}
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// ─── 辅助：证书 ───

func buildCertKey(source string, index int) string {
	return fmt.Sprintf("%s-%d", source, index)
}

func parseCertKey(key string) (string, int, error) {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("无效的证书 key: %s", key)
	}
	source := parts[0]
	if source != CertSourceFile && source != CertSourcePEM && source != CertSourceAutomate {
		return "", 0, fmt.Errorf("不支持的证书来源: %s", source)
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil || idx < 0 {
		return "", 0, fmt.Errorf("无效的证书下标: %s", parts[1])
	}
	return source, idx, nil
}

// validateCertForm 校验证书表单
//
// create=true 时强制要求私钥非空；
// create=false（更新）时私钥留空表示保留原值，允许通过。
func validateCertForm(req CertForm, create bool) error {
	switch req.Source {
	case CertSourceFile:
		if strings.TrimSpace(req.Certificate) == "" {
			return fmt.Errorf("证书路径不能为空")
		}
		if create && strings.TrimSpace(req.KeyContent) == "" {
			return fmt.Errorf("私钥路径不能为空")
		}
	case CertSourcePEM:
		if strings.TrimSpace(req.Certificate) == "" {
			return fmt.Errorf("证书 PEM 内容不能为空")
		}
		if create && strings.TrimSpace(req.KeyContent) == "" {
			return fmt.Errorf("私钥 PEM 内容不能为空")
		}
	case CertSourceAutomate:
		if strings.TrimSpace(req.Subject) == "" {
			return fmt.Errorf("自动签发主机名不能为空")
		}
	default:
		return fmt.Errorf("不支持的证书来源: %s", req.Source)
	}
	return nil
}

func ensureTLS(cfg *pkgcaddy.Config) *pkgcaddy.TLSApp {
	if cfg.Apps == nil {
		cfg.Apps = &pkgcaddy.AppsConfig{}
	}
	if cfg.Apps.TLS == nil {
		cfg.Apps.TLS = &pkgcaddy.TLSApp{}
	}
	return cfg.Apps.TLS
}

func ensureCerts(tls *pkgcaddy.TLSApp) *pkgcaddy.TLSCerts {
	if tls.Certificates == nil {
		tls.Certificates = &pkgcaddy.TLSCerts{}
	}
	return tls.Certificates
}

// appendAutomateSubject 把 subject 追加到自动签发列表
//
// 如果 automation.policies 为空，创建一个新策略；否则追加到第一个策略
func appendAutomateSubject(tls *pkgcaddy.TLSApp, subject string) {
	if tls.Automation == nil {
		tls.Automation = &pkgcaddy.TLSAutomation{}
	}
	if len(tls.Automation.Policies) == 0 {
		tls.Automation.Policies = []pkgcaddy.TLSPolicy{{Subjects: []string{subject}}}
		return
	}
	tls.Automation.Policies[0].Subjects = append(tls.Automation.Policies[0].Subjects, subject)
}

// replaceAutomateSubject 按全局 index 替换 subject，返回是否成功
func replaceAutomateSubject(tls *pkgcaddy.TLSApp, index int, subject string) bool {
	if tls.Automation == nil {
		return false
	}
	cur := 0
	for pi := range tls.Automation.Policies {
		policy := &tls.Automation.Policies[pi]
		for si := range policy.Subjects {
			if cur == index {
				policy.Subjects[si] = subject
				return true
			}
			cur++
		}
	}
	return false
}

// removeAutomateSubject 按全局 index 删除 subject，返回是否成功
func removeAutomateSubject(tls *pkgcaddy.TLSApp, index int) bool {
	if tls.Automation == nil {
		return false
	}
	cur := 0
	for pi := range tls.Automation.Policies {
		policy := &tls.Automation.Policies[pi]
		for si := range policy.Subjects {
			if cur == index {
				policy.Subjects = append(policy.Subjects[:si], policy.Subjects[si+1:]...)
				return true
			}
			cur++
		}
	}
	return false
}

// pickSecretStr 新值非空时用新值，否则保留旧值（与 system/config.go pickSecret 逻辑一致）
func pickSecretStr(newVal, oldVal string) string {
	if strings.TrimSpace(newVal) != "" {
		return newVal
	}
	return oldVal
}
