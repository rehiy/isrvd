package caddy

import (
	"bytes"
	"encoding/json"
)

// 本文件提供常用 Caddy JSON 配置的强类型结构体（Typed Modules）。
// 命名采用 Caddy 官方 JSON Schema 中的字段名，未覆盖的字段
// 通过 Extras 字段透传，保证 GetConfig → Load 来回不丢字段。
//
// 文档参考：https://caddyserver.com/docs/json/

// ----- 顶层 -----

// Config Caddy 顶层配置
//
// Extras 用于保留未建模的顶层字段，Marshal/Unmarshal 时自动透传。
type Config struct {
	Admin   *AdminConfig   `json:"admin,omitempty"`
	Logging *LoggingConfig `json:"logging,omitempty"`
	Storage map[string]any `json:"storage,omitempty"`
	Apps    *AppsConfig    `json:"apps,omitempty"`
	ID      string         `json:"@id,omitempty"`

	// Extras 保存所有未识别的顶层字段；Marshal 时与已知字段合并输出
	Extras map[string]json.RawMessage `json:"-"`
}

// configKnownKeys 已建模的顶层 JSON key，用于分离 unknown fields
var configKnownKeys = map[string]struct{}{
	"admin": {}, "logging": {}, "storage": {}, "apps": {}, "@id": {},
}

// MarshalJSON 合并已知字段 + Extras 输出
func (c Config) MarshalJSON() ([]byte, error) {
	type alias Config // 借助类型别名避免递归
	return mergeKnownAndExtras(alias(c), c.Extras)
}

// UnmarshalJSON 解析时把未知字段收集到 Extras
func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*c = Config(known)

	extras, err := pickExtras(data, configKnownKeys)
	if err != nil {
		return err
	}
	c.Extras = extras
	return nil
}

// AdminConfig admin 端点配置
type AdminConfig struct {
	Disabled      bool             `json:"disabled,omitempty"`
	Listen        string           `json:"listen,omitempty"`
	EnforceOrigin bool             `json:"enforce_origin,omitempty"`
	Origins       []string         `json:"origins,omitempty"`
	Config        *AdminAutoConfig `json:"config,omitempty"`
}

// AdminAutoConfig admin.config，例如 persist 持久化
type AdminAutoConfig struct {
	Persist *bool `json:"persist,omitempty"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Sink *LogSink         `json:"sink,omitempty"`
	Logs map[string]*Log  `json:"logs,omitempty"`
	ID   string           `json:"@id,omitempty"`
}

// LogSink 全局 sink
type LogSink struct {
	Writer map[string]any `json:"writer,omitempty"`
}

// Log 单个 logger
type Log struct {
	Writer            map[string]any `json:"writer,omitempty"`
	Encoder           map[string]any `json:"encoder,omitempty"`
	Level             string         `json:"level,omitempty"`
	Sampling          map[string]any `json:"sampling,omitempty"`
	Include           []string       `json:"include,omitempty"`
	Exclude           []string       `json:"exclude,omitempty"`
}

// AppsConfig 应用集合
//
// Extras 透传其他 app（layer4 / dynamic_dns / 第三方等）。
type AppsConfig struct {
	HTTP *HTTPApp `json:"http,omitempty"`
	TLS  *TLSApp  `json:"tls,omitempty"`
	PKI  *PKIApp  `json:"pki,omitempty"`

	Extras map[string]json.RawMessage `json:"-"`
}

// appsKnownKeys 已建模的 apps JSON key
var appsKnownKeys = map[string]struct{}{
	"http": {}, "tls": {}, "pki": {},
}

// MarshalJSON 合并已知字段 + Extras
func (a AppsConfig) MarshalJSON() ([]byte, error) {
	type alias AppsConfig
	return mergeKnownAndExtras(alias(a), a.Extras)
}

// UnmarshalJSON 收集未知 app 到 Extras
func (a *AppsConfig) UnmarshalJSON(data []byte) error {
	type alias AppsConfig
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*a = AppsConfig(known)

	extras, err := pickExtras(data, appsKnownKeys)
	if err != nil {
		return err
	}
	a.Extras = extras
	return nil
}

// ----- 内部辅助：unknown fields 透传 -----

// mergeKnownAndExtras 合并已知字段与 extras 输出 JSON 对象
//
// 实现技巧：直接在已知 JSON 的末尾 `}` 之前插入 extras 字段，
// 避免再次 unmarshal/marshal 整个对象。
func mergeKnownAndExtras(known any, extras map[string]json.RawMessage) ([]byte, error) {
	knownRaw, err := json.Marshal(known)
	if err != nil {
		return nil, err
	}
	if len(extras) == 0 {
		return knownRaw, nil
	}
	// known 必为对象，否则不合并
	end := bytes.LastIndexByte(knownRaw, '}')
	if end < 0 {
		return knownRaw, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(knownRaw) + 64*len(extras))
	buf.Write(knownRaw[:end])
	hasKnown := end > 1 // {} 时 end == 1
	for k, v := range extras {
		if hasKnown {
			buf.WriteByte(',')
		}
		hasKnown = true
		key, _ := json.Marshal(k)
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// pickExtras 从原始 JSON 中挑出不在 known 集合里的字段
func pickExtras(data []byte, known map[string]struct{}) (map[string]json.RawMessage, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		// 非对象（null 等）忽略
		return nil, nil
	}
	var extras map[string]json.RawMessage
	for k, v := range all {
		if _, ok := known[k]; ok {
			continue
		}
		if extras == nil {
			extras = map[string]json.RawMessage{}
		}
		extras[k] = v
	}
	return extras, nil
}

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
	Listen            []string         `json:"listen,omitempty"`
	Routes            []Route          `json:"routes,omitempty"`
	Errors            *HTTPErrors      `json:"errors,omitempty"`
	TLSConnPolicies   []map[string]any `json:"tls_connection_policies,omitempty"`
	AutomaticHTTPS    *AutomaticHTTPS  `json:"automatic_https,omitempty"`
	StrictSNIHost     *bool            `json:"strict_sni_host,omitempty"`
	IdleTimeout       string           `json:"idle_timeout,omitempty"`
	ReadTimeout       string           `json:"read_timeout,omitempty"`
	WriteTimeout      string           `json:"write_timeout,omitempty"`
	MaxHeaderBytes    int              `json:"max_header_bytes,omitempty"`
	Logs              *ServerLogs      `json:"logs,omitempty"`
	Protocols         []string         `json:"protocols,omitempty"`
	Metrics           map[string]any   `json:"metrics,omitempty"`
	ID                string           `json:"@id,omitempty"`
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
	Group    string         `json:"group,omitempty"`
	Match    []MatchSet     `json:"match,omitempty"`
	Handle   []Handler      `json:"handle,omitempty"`
	Terminal bool           `json:"terminal,omitempty"`
	ID       string         `json:"@id,omitempty"`
}

// MatchSet 匹配条件，所有字段为 AND；不同 MatchSet 之间为 OR
// 字段使用 Caddy 模块名 → 任意 JSON
type MatchSet map[string]any

// Handler 处理器，按顺序执行
// 必须包含 "handler" 字段（模块名），其余字段视模块而定
type Handler map[string]any

// ----- 常用 Handler 类型助手 -----

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

// ----- TLS App -----

// TLSApp tls 应用配置
type TLSApp struct {
	Certificates  *TLSCerts                 `json:"certificates,omitempty"`
	Automation    *TLSAutomation            `json:"automation,omitempty"`
	SessionTickets map[string]any           `json:"session_tickets,omitempty"`
	Cache         map[string]any            `json:"cache,omitempty"`
	DNS           map[string]any            `json:"dns,omitempty"`
}

// TLSCerts certificates 字段
type TLSCerts struct {
	LoadFiles    []TLSLoadFile  `json:"load_files,omitempty"`
	LoadFolders  []string       `json:"load_folders,omitempty"`
	LoadPEM      []TLSLoadPEM   `json:"load_pem,omitempty"`
	Automate     []string       `json:"automate,omitempty"`
}

// TLSLoadFile 从磁盘加载证书
type TLSLoadFile struct {
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags,omitempty"`
	Format      string   `json:"format,omitempty"`
}

// TLSLoadPEM 从内联 PEM 加载证书
type TLSLoadPEM struct {
	Certificate string   `json:"certificate"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags,omitempty"`
}

// TLSAutomation 自动化配置
type TLSAutomation struct {
	Policies        []TLSPolicy    `json:"policies,omitempty"`
	OnDemand        map[string]any `json:"on_demand,omitempty"`
	OCSP            map[string]any `json:"ocsp,omitempty"`
	RenewInterval   string         `json:"renew_interval,omitempty"`
	OCSPInterval    string         `json:"ocsp_interval,omitempty"`
}

// TLSPolicy 自动化策略
type TLSPolicy struct {
	Subjects        []string         `json:"subjects,omitempty"`
	Issuers         []map[string]any `json:"issuers,omitempty"`
	KeyType         string           `json:"key_type,omitempty"`
	OnDemand        bool             `json:"on_demand,omitempty"`
	MustStaple      bool             `json:"must_staple,omitempty"`
	RenewalWindow   float64          `json:"renewal_window_ratio,omitempty"`
	StorageOverride map[string]any   `json:"storage,omitempty"`
}

// ----- PKI App -----

// PKIApp pki 应用配置
type PKIApp struct {
	CAs map[string]map[string]any `json:"certificate_authorities,omitempty"`
}
