package caddy

// 本文件定义 HTTP app 相关的配置结构体及常用 Handler/Match 助手函数。

// ----- HTTP App -----

// HTTPApp http 应用配置
type HTTPApp struct {
	HTTPPort    int                    `json:"http_port,omitempty"`    // HTTP 监听端口（默认 80）
	HTTPSPort   int                    `json:"https_port,omitempty"`   // HTTPS 监听端口（默认 443）
	GracePeriod string                 `json:"grace_period,omitempty"` // 优雅关闭等待时间（如 "5s"）
	Servers     map[string]*HTTPServer `json:"servers,omitempty"`      // Server 名称 → 配置
}

// HTTPServer 单个 server
type HTTPServer struct {
	Listen          []string         `json:"listen,omitempty"`                  // 监听地址列表，如 [":80", ":443"]
	Routes          []Route          `json:"routes,omitempty"`                  // 路由规则列表
	Errors          *HTTPErrors      `json:"errors,omitempty"`                  // 错误处理配置
	TLSConnPolicies []map[string]any `json:"tls_connection_policies,omitempty"` // TLS 连接策略列表
	AutomaticHTTPS  *AutomaticHTTPS  `json:"automatic_https,omitempty"`         // 自动 HTTPS 配置
	StrictSNIHost   *bool            `json:"strict_sni_host,omitempty"`         // 是否严格校验 SNI
	IdleTimeout     string           `json:"idle_timeout,omitempty"`            // 空闲超时（如 "5s"）
	ReadTimeout     string           `json:"read_timeout,omitempty"`            // 读取超时
	WriteTimeout    string           `json:"write_timeout,omitempty"`           // 写入超时
	MaxHeaderBytes  int              `json:"max_header_bytes,omitempty"`        // 最大请求头字节数
	Logs            *ServerLogs      `json:"logs,omitempty"`                    // 访问日志配置
	Protocols       []string         `json:"protocols,omitempty"`               // 启用的协议：h1|h2|h2c|h3
	Metrics         map[string]any   `json:"metrics,omitempty"`                 // 指标配置（如 Prometheus）
	ID              string           `json:"@id,omitempty"`                     // 配置 ID（用于引用）
}

// ServerLogs 访问日志配置
type ServerLogs struct {
	DefaultLoggerName string            `json:"default_logger_name,omitempty"` // 默认日志名
	LoggerNames       map[string]string `json:"logger_names,omitempty"`        // Host → 日志名映射
	SkipHosts         []string          `json:"skip_hosts,omitempty"`          // 跳过日志的 Host 列表
	SkipUnmappedHosts bool              `json:"skip_unmapped_hosts,omitempty"` // 是否跳过未映射 Host 的日志
}

// HTTPErrors 错误处理
type HTTPErrors struct {
	Routes []Route `json:"routes,omitempty"` // 错误处理的路由规则
}

// AutomaticHTTPS 自动 HTTPS
type AutomaticHTTPS struct {
	Disable           bool     `json:"disable,omitempty"`                    // 是否禁用自动 HTTPS
	DisableRedirects  bool     `json:"disable_redirects,omitempty"`          // 是否禁用 HTTP→HTTPS 重定向
	DisableCerts      bool     `json:"disable_certificates,omitempty"`       // 是否禁用自动证书申请
	Skip              []string `json:"skip,omitempty"`                       // 跳过自动 HTTPS 的 Host 列表
	SkipCerts         []string `json:"skip_certificates,omitempty"`          // 跳过自动证书的域名列表
	IgnoreLoadedCerts bool     `json:"ignore_loaded_certificates,omitempty"` // 是否忽略已加载的证书
}

// Route HTTP 路由
type Route struct {
	Group    string     `json:"group,omitempty"`    // 路由组名（同组路由聚合）
	Match    []MatchSet `json:"match,omitempty"`    // 匹配条件（多个 MatchSet 为 OR 关系）
	Handle   []Handler  `json:"handle,omitempty"`   // 处理器链
	Terminal bool       `json:"terminal,omitempty"` // 是否终止后续路由匹配
	ID       string     `json:"@id,omitempty"`      // 路由 ID（用于引用）
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
