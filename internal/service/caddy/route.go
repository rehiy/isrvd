package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	pkgcaddy "isrvd/pkgs/caddy"
)

// ─── 路由 CRUD ───

// MatchForm 简化的 match 编辑模型
type MatchForm struct {
	Hosts    []string            `json:"hosts,omitempty"`
	Paths    []string            `json:"paths,omitempty"`
	Methods  []string            `json:"methods,omitempty"`
	Headers  map[string][]string `json:"headers,omitempty"`  // 按请求头匹配，key 为头字段名，value 为匹配值列表
	Protocol string              `json:"protocol,omitempty"` // 匹配协议：http / https
}

// HeaderOp 单条请求头/响应头操作
type HeaderOp struct {
	Op    string `json:"op"`    // set / add / delete
	Field string `json:"field"` // 头字段名，如 X-Real-IP
	Value string `json:"value"` // 值（delete 时留空）
}

// RewriteForm 简化的 URI rewrite 配置。
type RewriteForm struct {
	Method              string `json:"method,omitempty"`              // 转发前改写请求方法
	RewriteURI          string `json:"rewriteUri,omitempty"`          // 完整 URI 替换（支持 Caddy 占位符）
	StripPathPrefix     string `json:"stripPathPrefix,omitempty"`     // 去掉路径前缀
	StripPathSuffix     string `json:"stripPathSuffix,omitempty"`     // 去掉路径后缀
	URISubstringFind    string `json:"uriSubstringFind,omitempty"`    // 子串查找（配合 uriSubstringReplace）
	URISubstringReplace string `json:"uriSubstringReplace,omitempty"` // 子串替换
}

// HandlerForm 简化的 handler 编辑模型，按 Kind 解释字段
type HandlerForm struct {
	Kind string `json:"kind"` // reverse_proxy / file_server / static_response / rewrite / raw

	// reverse_proxy
	Upstreams    []string     `json:"upstreams,omitempty"`
	FastCGI      bool         `json:"fastcgi,omitempty"`      // 启用 FastCGI 传输协议（PHP-FPM 等）
	FastCGIRoot  string       `json:"fastcgiRoot,omitempty"`  // FastCGI 文档根目录
	DialTimeout  string       `json:"dialTimeout,omitempty"`  // 连接上游超时，如 10s
	ReadTimeout  string       `json:"readTimeout,omitempty"`  // 读取上游响应超时，如 30s
	WriteTimeout string       `json:"writeTimeout,omitempty"` // 向上游写入请求超时，如 30s
	ProxyRewrite *RewriteForm `json:"proxyRewrite,omitempty"` // reverse_proxy.rewrite 配置

	// file_server
	Root   string `json:"root,omitempty"`
	Browse bool   `json:"browse,omitempty"`

	// static_response
	StatusCode int    `json:"statusCode,omitempty"`
	Body       string `json:"body,omitempty"`

	// rewrite：URI 重写
	RewriteURI          string `json:"rewriteUri,omitempty"`          // 完整 URI 替换（支持 Caddy 占位符）
	StripPathPrefix     string `json:"stripPathPrefix,omitempty"`     // 去掉路径前缀
	StripPathSuffix     string `json:"stripPathSuffix,omitempty"`     // 去掉路径后缀
	URISubstringFind    string `json:"uriSubstringFind,omitempty"`    // 子串查找（配合 uriSubstringReplace）
	URISubstringReplace string `json:"uriSubstringReplace,omitempty"` // 子串替换

	// headers 中间件：请求头/响应头操作（可附加在任意终止型 handler 上，生成前置 headers handler）
	RequestHeaders  []HeaderOp `json:"requestHeaders,omitempty"`  // 请求头操作列表
	ResponseHeaders []HeaderOp `json:"responseHeaders,omitempty"` // 响应头操作列表

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
	case HandlerKindRewrite:
		if req.Handler.RewriteURI == "" &&
			req.Handler.StripPathPrefix == "" &&
			req.Handler.StripPathSuffix == "" &&
			req.Handler.URISubstringFind == "" {
			return fmt.Errorf("rewrite 至少需要填写一个重写规则")
		}
	case HandlerKindRaw:
		// raw 模式下不允许同时设置 requestHeaders/responseHeaders（直接在 raw JSON 中编辑）
		if len(req.Handler.RequestHeaders) > 0 || len(req.Handler.ResponseHeaders) > 0 {
			return fmt.Errorf("raw 模式下请直接在 JSON 中编辑 headers handler")
		}
		if len(req.Handler.Raw) == 0 {
			return fmt.Errorf("原始 handle 不能为空")
		}
	default:
		return fmt.Errorf("不支持的 handler 类型: %s", req.Handler.Kind)
	}
	return nil
}

func hasRewriteForm(r *RewriteForm) bool {
	return r != nil && (strings.TrimSpace(r.Method) != "" ||
		strings.TrimSpace(r.RewriteURI) != "" ||
		strings.TrimSpace(r.StripPathPrefix) != "" ||
		strings.TrimSpace(r.StripPathSuffix) != "" ||
		strings.TrimSpace(r.URISubstringFind) != "")
}

func buildRewriteMap(r *RewriteForm) map[string]any {
	h := map[string]any{}
	if r == nil {
		return h
	}
	if r.Method != "" {
		h["method"] = r.Method
	}
	if r.RewriteURI != "" {
		h["uri"] = r.RewriteURI
	}
	if r.StripPathPrefix != "" {
		h["strip_path_prefix"] = r.StripPathPrefix
	}
	if r.StripPathSuffix != "" {
		h["strip_path_suffix"] = r.StripPathSuffix
	}
	if r.URISubstringFind != "" {
		h["uri_substring"] = []map[string]any{{
			"find":    r.URISubstringFind,
			"replace": r.URISubstringReplace,
		}}
	}
	return h
}

func parseRewriteMap(h map[string]any) *RewriteForm {
	out := &RewriteForm{}
	out.Method, _ = h["method"].(string)
	out.RewriteURI, _ = h["uri"].(string)
	out.StripPathPrefix, _ = h["strip_path_prefix"].(string)
	out.StripPathSuffix, _ = h["strip_path_suffix"].(string)
	if subs, ok := h["uri_substring"].([]any); ok && len(subs) > 0 {
		if sub, ok := subs[0].(map[string]any); ok {
			out.URISubstringFind, _ = sub["find"].(string)
			out.URISubstringReplace, _ = sub["replace"].(string)
		}
	}
	if !hasRewriteForm(out) {
		return nil
	}
	return out
}

// buildHeadersOps 将 HeaderOp 列表转换为 Caddy headers handler 的 set/add/delete 结构
func buildHeadersOps(ops []HeaderOp) map[string]any {
	set := map[string][]string{}
	add := map[string][]string{}
	del := []string{}
	for _, op := range ops {
		switch op.Op {
		case "set":
			set[op.Field] = []string{op.Value}
		case "add":
			add[op.Field] = append(add[op.Field], op.Value)
		case "delete":
			del = append(del, op.Field)
		}
	}
	out := map[string]any{}
	if len(set) > 0 {
		out["set"] = set
	}
	if len(add) > 0 {
		out["add"] = add
	}
	if len(del) > 0 {
		out["delete"] = del
	}
	return out
}

// parseHeadersOps 从 Caddy headers handler 的 set/add/delete 结构解析为 HeaderOp 列表
func parseHeadersOps(m map[string]any) []HeaderOp {
	var ops []HeaderOp
	if setMap, ok := m["set"].(map[string]any); ok {
		for k, v := range setMap {
			val := ""
			if arr, ok := v.([]any); ok && len(arr) > 0 {
				val, _ = arr[0].(string)
			} else if s, ok := v.(string); ok {
				val = s
			}
			ops = append(ops, HeaderOp{Op: "set", Field: k, Value: val})
		}
	}
	if addMap, ok := m["add"].(map[string]any); ok {
		for k, v := range addMap {
			if arr, ok := v.([]any); ok {
				for _, item := range arr {
					val, _ := item.(string)
					ops = append(ops, HeaderOp{Op: "add", Field: k, Value: val})
				}
			}
		}
	}
	if delArr, ok := m["delete"].([]any); ok {
		for _, item := range delArr {
			if field, ok := item.(string); ok {
				ops = append(ops, HeaderOp{Op: "delete", Field: field})
			}
		}
	}
	return ops
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
		if len(req.Match.Headers) > 0 {
			match["header"] = req.Match.Headers
		}
		if req.Match.Protocol != "" {
			match["protocol"] = req.Match.Protocol
		}
		if len(match) > 0 {
			r.Match = []pkgcaddy.MatchSet{match}
		}
	}

	// 构建终止型 handler
	var terminalHandlers []pkgcaddy.Handler
	switch req.Handler.Kind {
	case HandlerKindReverseProxy:
		h := pkgcaddy.HandlerReverseProxy(nonEmpty(req.Handler.Upstreams)...)
		if req.Handler.FastCGI {
			transport := map[string]any{"protocol": "fastcgi"}
			if req.Handler.FastCGIRoot != "" {
				transport["root"] = req.Handler.FastCGIRoot
			}
			h["transport"] = transport
		} else if req.Handler.DialTimeout != "" || req.Handler.ReadTimeout != "" || req.Handler.WriteTimeout != "" {
			transport := map[string]any{"protocol": "http"}
			if req.Handler.DialTimeout != "" {
				transport["dial_timeout"] = req.Handler.DialTimeout
			}
			if req.Handler.ReadTimeout != "" {
				transport["response_header_timeout"] = req.Handler.ReadTimeout
			}
			if req.Handler.WriteTimeout != "" {
				transport["write_timeout"] = req.Handler.WriteTimeout
			}
			h["transport"] = transport
		}
		if hasRewriteForm(req.Handler.ProxyRewrite) {
			h["rewrite"] = buildRewriteMap(req.Handler.ProxyRewrite)
		}
		terminalHandlers = []pkgcaddy.Handler{h}
	case HandlerKindFileServer:
		terminalHandlers = []pkgcaddy.Handler{pkgcaddy.HandlerFileServer(req.Handler.Root, req.Handler.Browse)}
	case HandlerKindStaticResp:
		terminalHandlers = []pkgcaddy.Handler{pkgcaddy.HandlerStaticResponse(req.Handler.StatusCode, req.Handler.Body)}
	case HandlerKindRewrite:
		h := pkgcaddy.Handler{"handler": "rewrite"}
		for k, v := range buildRewriteMap(&RewriteForm{
			RewriteURI:          req.Handler.RewriteURI,
			StripPathPrefix:     req.Handler.StripPathPrefix,
			StripPathSuffix:     req.Handler.StripPathSuffix,
			URISubstringFind:    req.Handler.URISubstringFind,
			URISubstringReplace: req.Handler.URISubstringReplace,
		}) {
			h[k] = v
		}
		terminalHandlers = []pkgcaddy.Handler{h}
	case HandlerKindRaw:
		var handlers []pkgcaddy.Handler
		if err := json.Unmarshal(req.Handler.Raw, &handlers); err != nil {
			return r, fmt.Errorf("原始 handle 解析失败: %w", err)
		}
		r.Handle = handlers
		return r, nil
	}

	// 如果有 headers 中间件，插入到终止 handler 前面
	if len(req.Handler.RequestHeaders) > 0 || len(req.Handler.ResponseHeaders) > 0 {
		hh := pkgcaddy.Handler{"handler": "headers"}
		if len(req.Handler.RequestHeaders) > 0 {
			hh["request"] = buildHeadersOps(req.Handler.RequestHeaders)
		}
		if len(req.Handler.ResponseHeaders) > 0 {
			hh["response"] = buildHeadersOps(req.Handler.ResponseHeaders)
		}
		r.Handle = append([]pkgcaddy.Handler{hh}, terminalHandlers...)
	} else {
		r.Handle = terminalHandlers
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
		// header 匹配
		if hdr, ok := m["header"]; ok {
			switch v := hdr.(type) {
			case map[string][]string:
				mf.Headers = v
			case map[string]any:
				mf.Headers = map[string][]string{}
				for k, val := range v {
					mf.Headers[k] = toStrSlice(val)
				}
			}
		}
		// protocol 匹配
		mf.Protocol, _ = m["protocol"].(string)
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

	// 识别 [headers, <终止handler>] 组合：提取 headers 中间件
	var headersHandler pkgcaddy.Handler
	workHandlers := handlers
	if len(handlers) == 2 {
		if k, _ := handlers[0]["handler"].(string); k == HandlerKindHeaders {
			headersHandler = handlers[0]
			workHandlers = handlers[1:]
		}
	}

	if len(workHandlers) > 1 {
		return rawHandler(handlers)
	}

	h := workHandlers[0]
	kind, _ := h["handler"].(string)

	var form *HandlerForm
	switch kind {
	case HandlerKindReverseProxy:
		form = &HandlerForm{Kind: kind, Upstreams: extractUpstreams(h)}
		if rw, ok := h["rewrite"].(map[string]any); ok {
			form.ProxyRewrite = parseRewriteMap(rw)
		}
		if t, ok := h["transport"].(map[string]any); ok {
			if proto, _ := t["protocol"].(string); proto == "fastcgi" {
				form.FastCGI = true
				form.FastCGIRoot, _ = t["root"].(string)
			} else {
				form.DialTimeout, _ = t["dial_timeout"].(string)
				form.ReadTimeout, _ = t["response_header_timeout"].(string)
				form.WriteTimeout, _ = t["write_timeout"].(string)
			}
		}
	case HandlerKindFileServer:
		root, _ := h["root"].(string)
		_, browse := h["browse"]
		form = &HandlerForm{Kind: kind, Root: root, Browse: browse}
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
		form = &HandlerForm{Kind: kind, StatusCode: status, Body: body}
	case HandlerKindRewrite:
		form = &HandlerForm{Kind: kind}
		if rw := parseRewriteMap(h); rw != nil {
			form.RewriteURI = rw.RewriteURI
			form.StripPathPrefix = rw.StripPathPrefix
			form.StripPathSuffix = rw.StripPathSuffix
			form.URISubstringFind = rw.URISubstringFind
			form.URISubstringReplace = rw.URISubstringReplace
		}
	default:
		return rawHandler(handlers)
	}

	// 将 headers 中间件字段附加到表单
	if headersHandler != nil {
		if req, ok := headersHandler["request"].(map[string]any); ok {
			form.RequestHeaders = parseHeadersOps(req)
		}
		if resp, ok := headersHandler["response"].(map[string]any); ok {
			form.ResponseHeaders = parseHeadersOps(resp)
		}
	}
	return form
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
