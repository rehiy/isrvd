package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// 本文件定义 HTTP app 相关的配置结构体及 Basic Auth 助手函数。

const (
	durationNumberPrefix = "\x00caddy-duration-number:"
	durationStringPrefix = "\x00caddy-duration-string:"
)

var (
	httpAppKnownKeys = jsonKeySet(
		"http_port", "https_port", "grace_period", "shutdown_delay", "servers",
	)
	httpServerKnownKeys = jsonKeySet(
		"listen", "listener_wrappers", "read_timeout", "read_header_timeout", "write_timeout",
		"idle_timeout", "keepalive_interval", "max_header_bytes", "enable_full_duplex", "routes",
		"errors", "named_routes", "tls_connection_policies", "automatic_https", "strict_sni_host",
		"trusted_proxies", "client_ip_headers", "trusted_proxies_strict", "logs", "protocols",
		"metrics", "@id",
	)
	serverLogsKnownKeys = jsonKeySet(
		"default_logger_name", "logger_names", "skip_hosts", "skip_unmapped_hosts",
		"should_log_credentials", "trace",
	)
	httpErrorsKnownKeys     = jsonKeySet("routes")
	automaticHTTPSKnownKeys = jsonKeySet(
		"disable", "disable_redirects", "disable_certificates", "skip", "skip_certificates",
		"ignore_loaded_certificates",
	)
	routeKnownKeys = jsonKeySet("group", "match", "handle", "terminal", "@id")
)

// ----- HTTP App -----

// Duration 是 Caddy JSON duration，兼容字符串和纳秒数字两种形式。
//
// String 返回适合表单展示的 duration；数字形式会转换为 Go duration 文本。
type Duration string

// MarshalJSON 保留从 Caddy 读取到的原始字符串或数字表示。
func (d Duration) MarshalJSON() ([]byte, error) {
	value, numeric := d.value()
	if !numeric {
		return json.Marshal(value)
	}
	var number json.Number
	if err := json.Unmarshal([]byte(value), &number); err != nil {
		return nil, fmt.Errorf("无效的 caddy duration 数字 %q: %w", value, err)
	}
	return []byte(value), nil
}

// UnmarshalJSON 接受 duration 字符串或纳秒数字。
func (d *Duration) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*d = ""
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		*d = Duration(durationStringPrefix + value)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(trimmed, &number); err != nil {
		return fmt.Errorf("caddy duration 必须是字符串或数字: %w", err)
	}
	*d = Duration(durationNumberPrefix + string(trimmed))
	return nil
}

// String 返回 duration 的可读字符串。
func (d Duration) String() string {
	value, numeric := d.value()
	if !numeric {
		return value
	}
	var nanoseconds int64
	if err := json.Unmarshal([]byte(value), &nanoseconds); err == nil {
		return time.Duration(nanoseconds).String()
	}
	return value
}

// StringArray 是兼容单个 JSON 字符串和字符串数组的列表。
type StringArray []string

// UnmarshalJSON 接受 "logger" 或 ["logger-a", "logger-b"]。
func (s *StringArray) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*s = nil
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var single string
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*s = StringArray{single}
		return nil
	}
	var values []string
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return fmt.Errorf("logger_names 的值必须是字符串或字符串数组: %w", err)
	}
	*s = values
	return nil
}

// HTTPApp http 应用配置
type HTTPApp struct {
	HTTPPort      int                        `json:"http_port,omitempty"`      // HTTP 监听端口（默认 80）
	HTTPSPort     int                        `json:"https_port,omitempty"`     // HTTPS 监听端口（默认 443）
	GracePeriod   Duration                   `json:"grace_period,omitempty"`   // 优雅关闭等待时间（如 "5s"）
	ShutdownDelay Duration                   `json:"shutdown_delay,omitempty"` // 延迟关闭等待时间
	Servers       map[string]*HTTPServer     `json:"servers,omitempty"`        // Server 名称 → 配置
	Extras        map[string]json.RawMessage `json:"-"`                        // 未建模的 HTTP app 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (h HTTPApp) MarshalJSON() ([]byte, error) {
	type alias HTTPApp
	return mergeKnownAndExtras(alias(h), h.Extras)
}

// UnmarshalJSON 收集未建模的 HTTP app 配置。
func (h *HTTPApp) UnmarshalJSON(data []byte) error {
	type alias HTTPApp
	known, extras, err := unmarshalKnownAndExtras[alias](data, httpAppKnownKeys)
	if err != nil {
		return err
	}
	*h = HTTPApp(known)
	h.Extras = extras
	return nil
}

// HTTPServer 单个 server
type HTTPServer struct {
	Listen               []string                   `json:"listen,omitempty"`                  // 监听地址列表，如 [":80", ":443"]
	ListenerWrappers     []json.RawMessage          `json:"listener_wrappers,omitempty"`       // listener wrapper 模块配置
	ReadTimeout          Duration                   `json:"read_timeout,omitempty"`            // 读取超时
	ReadHeaderTimeout    Duration                   `json:"read_header_timeout,omitempty"`     // 请求头读取超时
	WriteTimeout         Duration                   `json:"write_timeout,omitempty"`           // 写入超时
	IdleTimeout          Duration                   `json:"idle_timeout,omitempty"`            // 空闲超时
	KeepAliveInterval    Duration                   `json:"keepalive_interval,omitempty"`      // TCP keepalive 间隔
	MaxHeaderBytes       int                        `json:"max_header_bytes,omitempty"`        // 最大请求头字节数
	EnableFullDuplex     bool                       `json:"enable_full_duplex,omitempty"`      // 启用 HTTP/1 全双工
	Routes               []Route                    `json:"routes,omitempty"`                  // 路由规则列表
	Errors               *HTTPErrors                `json:"errors,omitempty"`                  // 错误处理配置
	NamedRoutes          map[string]*Route          `json:"named_routes,omitempty"`            // 命名路由
	TLSConnPolicies      []map[string]any           `json:"tls_connection_policies,omitempty"` // TLS 连接策略列表
	AutomaticHTTPS       *AutomaticHTTPS            `json:"automatic_https,omitempty"`         // 自动 HTTPS 配置
	StrictSNIHost        *bool                      `json:"strict_sni_host,omitempty"`         // 是否严格校验 SNI
	TrustedProxies       json.RawMessage            `json:"trusted_proxies,omitempty"`         // 可信代理源模块
	ClientIPHeaders      []string                   `json:"client_ip_headers,omitempty"`       // 获取客户端 IP 的请求头
	TrustedProxiesStrict int                        `json:"trusted_proxies_strict,omitempty"`  // 可信代理头解析顺序
	Logs                 *ServerLogs                `json:"logs,omitempty"`                    // 访问日志配置
	Protocols            []string                   `json:"protocols,omitempty"`               // 启用的协议：h1|h2|h2c|h3
	Metrics              map[string]any             `json:"metrics,omitempty"`                 // 指标配置（如 Prometheus）
	ID                   ID                         `json:"@id,omitempty"`                     // 配置 ID（用于引用）
	Extras               map[string]json.RawMessage `json:"-"`                                 // 未建模的 server 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (h HTTPServer) MarshalJSON() ([]byte, error) {
	type alias HTTPServer
	return mergeKnownAndExtras(alias(h), h.Extras)
}

// UnmarshalJSON 收集未建模的 server 配置。
func (h *HTTPServer) UnmarshalJSON(data []byte) error {
	type alias HTTPServer
	known, extras, err := unmarshalKnownAndExtras[alias](data, httpServerKnownKeys)
	if err != nil {
		return err
	}
	*h = HTTPServer(known)
	h.Extras = extras
	return nil
}

// ServerLogs 访问日志配置
type ServerLogs struct {
	DefaultLoggerName    string                     `json:"default_logger_name,omitempty"`    // 默认日志名
	LoggerNames          map[string]StringArray     `json:"logger_names,omitempty"`           // Host → 日志名列表映射
	SkipHosts            []string                   `json:"skip_hosts,omitempty"`             // 跳过日志的 Host 列表
	SkipUnmappedHosts    bool                       `json:"skip_unmapped_hosts,omitempty"`    // 是否跳过未映射 Host 的日志
	ShouldLogCredentials bool                       `json:"should_log_credentials,omitempty"` // 是否记录凭据请求头
	Trace                bool                       `json:"trace,omitempty"`                  // 是否记录 handler trace
	Extras               map[string]json.RawMessage `json:"-"`                                // 未建模的访问日志配置
}

// MarshalJSON 合并已知字段 + Extras。
func (s ServerLogs) MarshalJSON() ([]byte, error) {
	type alias ServerLogs
	return mergeKnownAndExtras(alias(s), s.Extras)
}

// UnmarshalJSON 收集未建模的访问日志配置。
func (s *ServerLogs) UnmarshalJSON(data []byte) error {
	type alias ServerLogs
	known, extras, err := unmarshalKnownAndExtras[alias](data, serverLogsKnownKeys)
	if err != nil {
		return err
	}
	*s = ServerLogs(known)
	s.Extras = extras
	return nil
}

// HTTPErrors 错误处理
type HTTPErrors struct {
	Routes []Route                    `json:"routes,omitempty"` // 错误处理的路由规则
	Extras map[string]json.RawMessage `json:"-"`                // 未建模的错误处理配置
}

// MarshalJSON 合并已知字段 + Extras。
func (h HTTPErrors) MarshalJSON() ([]byte, error) {
	type alias HTTPErrors
	return mergeKnownAndExtras(alias(h), h.Extras)
}

// UnmarshalJSON 收集未建模的错误处理配置。
func (h *HTTPErrors) UnmarshalJSON(data []byte) error {
	type alias HTTPErrors
	known, extras, err := unmarshalKnownAndExtras[alias](data, httpErrorsKnownKeys)
	if err != nil {
		return err
	}
	*h = HTTPErrors(known)
	h.Extras = extras
	return nil
}

// AutomaticHTTPS 自动 HTTPS
type AutomaticHTTPS struct {
	Disable           bool                       `json:"disable,omitempty"`                    // 是否禁用自动 HTTPS
	DisableRedirects  bool                       `json:"disable_redirects,omitempty"`          // 是否禁用 HTTP→HTTPS 重定向
	DisableCerts      bool                       `json:"disable_certificates,omitempty"`       // 是否禁用自动证书申请
	Skip              []string                   `json:"skip,omitempty"`                       // 跳过自动 HTTPS 的 Host 列表
	SkipCerts         []string                   `json:"skip_certificates,omitempty"`          // 跳过自动证书的域名列表
	IgnoreLoadedCerts bool                       `json:"ignore_loaded_certificates,omitempty"` // 是否忽略已加载的证书
	Extras            map[string]json.RawMessage `json:"-"`                                    // 未建模的自动 HTTPS 配置
}

// MarshalJSON 合并已知字段 + Extras。
func (a AutomaticHTTPS) MarshalJSON() ([]byte, error) {
	type alias AutomaticHTTPS
	return mergeKnownAndExtras(alias(a), a.Extras)
}

// UnmarshalJSON 收集未建模的自动 HTTPS 配置。
func (a *AutomaticHTTPS) UnmarshalJSON(data []byte) error {
	type alias AutomaticHTTPS
	known, extras, err := unmarshalKnownAndExtras[alias](data, automaticHTTPSKnownKeys)
	if err != nil {
		return err
	}
	*a = AutomaticHTTPS(known)
	a.Extras = extras
	return nil
}

// Route HTTP 路由
type Route struct {
	Group    string                     `json:"group,omitempty"`    // 路由组名（同组路由聚合）
	Match    []MatchSet                 `json:"match,omitempty"`    // 匹配条件（多个 MatchSet 为 OR 关系）
	Handle   []Handler                  `json:"handle,omitempty"`   // 处理器链
	Terminal bool                       `json:"terminal,omitempty"` // 是否终止后续路由匹配
	ID       ID                         `json:"@id,omitempty"`      // 路由 ID（用于引用）
	Extras   map[string]json.RawMessage `json:"-"`                  // 未建模的路由配置
}

// MarshalJSON 合并已知字段 + Extras。
func (r Route) MarshalJSON() ([]byte, error) {
	type alias Route
	return mergeKnownAndExtras(alias(r), r.Extras)
}

// UnmarshalJSON 收集未建模的路由配置。
func (r *Route) UnmarshalJSON(data []byte) error {
	type alias Route
	known, extras, err := unmarshalKnownAndExtras[alias](data, routeKnownKeys)
	if err != nil {
		return err
	}
	*r = Route(known)
	r.Extras = extras
	return nil
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

// ─── 辅助函数 ───

func (d Duration) value() (string, bool) {
	value := string(d)
	if len(value) >= len(durationNumberPrefix) && value[:len(durationNumberPrefix)] == durationNumberPrefix {
		return value[len(durationNumberPrefix):], true
	}
	if len(value) >= len(durationStringPrefix) && value[:len(durationStringPrefix)] == durationStringPrefix {
		return value[len(durationStringPrefix):], false
	}
	return value, false
}
