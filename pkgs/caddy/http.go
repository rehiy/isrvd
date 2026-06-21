package caddy

// 本文件定义 HTTP app 相关的配置结构体及 Basic Auth 助手函数。

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

// ----- Basic Auth -----

// BasicAuthAccount basic_auth 单个账号（密码为 bcrypt hash）
type BasicAuthAccount struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
	Salt     string `json:"salt,omitempty"`
}

// HandlerBasicAuth 构造 authentication handler（http_basic provider）
func HandlerBasicAuth(realm string, accounts []BasicAuthAccount) Handler {
	provider := map[string]any{"accounts": accounts}
	if realm != "" {
		provider["realm"] = realm
	}
	return Handler{
		"handler":   "authentication",
		"providers": map[string]any{"http_basic": provider},
	}
}

// BasicAuthFromHandler 从 Handler 中提取 basic_auth 账号列表，失败返回 nil
func BasicAuthFromHandler(h Handler) (realm string, accounts []BasicAuthAccount, ok bool) {
	if h["handler"] != "authentication" {
		return
	}
	providers, _ := h["providers"].(map[string]any)
	if providers == nil {
		return
	}
	basic, _ := providers["http_basic"].(map[string]any)
	if basic == nil {
		return
	}
	realm, _ = basic["realm"].(string)
	accsRaw, _ := basic["accounts"].([]any)
	for _, raw := range accsRaw {
		m, _ := raw.(map[string]any)
		if m == nil {
			continue
		}
		username, _ := m["username"].(string)
		password, _ := m["password"].(string)
		salt, _ := m["salt"].(string)
		if username != "" {
			accounts = append(accounts, BasicAuthAccount{Username: username, Password: password, Salt: salt})
		}
	}
	ok = true
	return
}
