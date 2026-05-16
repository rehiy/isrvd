package caddy

// 本文件定义 HTTP app 相关的配置结构体及常用 Handler/Match 助手函数。

// ----- HTTP App -----

// HTTPApp http 应用配置
type HTTPApp struct {
	HTTPPort    int                    `json:"http_port,omitempty"`
	HTTPSPort   int                    `json:"https_port,omitempty"`
	GracePeriod string                 `json:"grace_period,omitempty"`
	Servers     map[string]*HTTPServer `json:"servers,omitempty"`
}

// HTTPServer 单个 server
type HTTPServer struct {
	Listen          []string         `json:"listen,omitempty"`
	Routes          []Route          `json:"routes,omitempty"`
	Errors          *HTTPErrors      `json:"errors,omitempty"`
	TLSConnPolicies []map[string]any `json:"tls_connection_policies,omitempty"`
	AutomaticHTTPS  *AutomaticHTTPS  `json:"automatic_https,omitempty"`
	StrictSNIHost   *bool            `json:"strict_sni_host,omitempty"`
	IdleTimeout     string           `json:"idle_timeout,omitempty"`
	ReadTimeout     string           `json:"read_timeout,omitempty"`
	WriteTimeout    string           `json:"write_timeout,omitempty"`
	MaxHeaderBytes  int              `json:"max_header_bytes,omitempty"`
	Logs            *ServerLogs      `json:"logs,omitempty"`
	Protocols       []string         `json:"protocols,omitempty"`
	Metrics         map[string]any   `json:"metrics,omitempty"`
	ID              string           `json:"@id,omitempty"`
}

// ServerLogs 访问日志配置
type ServerLogs struct {
	DefaultLoggerName string            `json:"default_logger_name,omitempty"`
	LoggerNames       map[string]string `json:"logger_names,omitempty"`
	SkipHosts         []string          `json:"skip_hosts,omitempty"`
	SkipUnmappedHosts bool              `json:"skip_unmapped_hosts,omitempty"`
}

// HTTPErrors 错误处理
type HTTPErrors struct {
	Routes []Route `json:"routes,omitempty"`
}

// AutomaticHTTPS 自动 HTTPS
type AutomaticHTTPS struct {
	Disable           bool     `json:"disable,omitempty"`
	DisableRedirects  bool     `json:"disable_redirects,omitempty"`
	DisableCerts      bool     `json:"disable_certificates,omitempty"`
	Skip              []string `json:"skip,omitempty"`
	SkipCerts         []string `json:"skip_certificates,omitempty"`
	IgnoreLoadedCerts bool     `json:"ignore_loaded_certificates,omitempty"`
}

// Route HTTP 路由
type Route struct {
	Group    string     `json:"group,omitempty"`
	Match    []MatchSet `json:"match,omitempty"`
	Handle   []Handler  `json:"handle,omitempty"`
	Terminal bool       `json:"terminal,omitempty"`
	ID       string     `json:"@id,omitempty"`
}

// MatchSet 匹配条件，所有字段为 AND；不同 MatchSet 之间为 OR
// 字段使用 Caddy 模块名 → 任意 JSON
type MatchSet map[string]any

// Handler 处理器，按顺序执行
// 必须包含 "handler" 字段（模块名），其余字段视模块而定
type Handler map[string]any

// ----- 常用 Handler 助手 -----

// HandlerStaticResponse 构造 static_response handler
func HandlerStaticResponse(statusCode int, body string) Handler {
	h := Handler{"handler": "static_response"}
	if statusCode > 0 {
		h["status_code"] = statusCode
	}
	if body != "" {
		h["body"] = body
	}
	return h
}

// HandlerSubroute 构造 subroute handler
func HandlerSubroute(routes []Route) Handler {
	return Handler{"handler": "subroute", "routes": routes}
}

// HandlerHeaders 构造 headers handler，支持 set/add/delete
func HandlerHeaders(set, add map[string][]string, del []string) Handler {
	resp := map[string]any{}
	if len(set) > 0 {
		resp["set"] = set
	}
	if len(add) > 0 {
		resp["add"] = add
	}
	if len(del) > 0 {
		resp["delete"] = del
	}
	return Handler{
		"handler":  "headers",
		"response": resp,
	}
}

// HandlerRewrite URI 重写
func HandlerRewrite(stripPrefix, uri string) Handler {
	h := Handler{"handler": "rewrite"}
	if stripPrefix != "" {
		h["strip_path_prefix"] = stripPrefix
	}
	if uri != "" {
		h["uri"] = uri
	}
	return h
}

// HandlerReverseProxy 构造 reverse_proxy handler
func HandlerReverseProxy(upstreams ...string) Handler {
	ups := make([]map[string]any, 0, len(upstreams))
	for _, u := range upstreams {
		ups = append(ups, map[string]any{"dial": u})
	}
	return Handler{
		"handler":   "reverse_proxy",
		"upstreams": ups,
	}
}

// HandlerFileServer 构造 file_server handler
func HandlerFileServer(root string, browse bool) Handler {
	h := Handler{"handler": "file_server"}
	if root != "" {
		h["root"] = root
	}
	if browse {
		h["browse"] = map[string]any{}
	}
	return h
}

// HandlerEncode 启用响应压缩，例如 gzip / zstd
func HandlerEncode(encodings ...string) Handler {
	enc := map[string]any{}
	for _, e := range encodings {
		enc[e] = map[string]any{}
	}
	return Handler{
		"handler":   "encode",
		"encodings": enc,
	}
}

// ----- 常用 Match 助手 -----

// MatchHost 主机匹配
func MatchHost(hosts ...string) MatchSet {
	return MatchSet{"host": hosts}
}

// MatchPath 路径匹配
func MatchPath(paths ...string) MatchSet {
	return MatchSet{"path": paths}
}

// MatchMethod 方法匹配
func MatchMethod(methods ...string) MatchSet {
	return MatchSet{"method": methods}
}

// Merge 合并多个匹配条件到一个 MatchSet（AND 关系）
func Merge(sets ...MatchSet) MatchSet {
	out := MatchSet{}
	for _, s := range sets {
		for k, v := range s {
			out[k] = v
		}
	}
	return out
}
