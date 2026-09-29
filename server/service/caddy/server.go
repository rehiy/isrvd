package caddy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"isrvd/pkgs/caddy"
)

const serverIDPrefix = "~"

var (
	ErrServerNameInvalid   = errors.New("服务名称无效")
	ErrServerConfigInvalid = errors.New("服务配置无效")
	ErrServerNotFound      = errors.New("服务不存在")
	ErrServerExists        = errors.New("服务已存在")
	ErrDefaultServer       = errors.New("默认服务 srv0 不能通过服务接口删除")
	ErrLastServer          = errors.New("至少保留一个服务，无法删除")
)

// ServerAutomaticHTTPSForm 是服务接口允许编辑的 automatic_https 字段。
type ServerAutomaticHTTPSForm struct {
	Disable          bool `json:"disable"`           // 禁用自动 HTTPS（不再自动申请证书）
	DisableRedirects bool `json:"disable_redirects"` // 禁用 HTTP 到 HTTPS 自动跳转
}

// ServerForm 是服务接口允许编辑的 Caddy HTTP server 字段。
//
// 更新接口采用 PUT 语义：这些字段组成一个完整的可编辑视图；省略、null 或空值会
// 恢复相应的 Caddy 默认值。未出现在本结构体中的 routes、logs、errors、TLS 策略及
// 扩展字段始终由后端从现有配置合并保留。
type ServerForm struct {
	Listen         []string                  `json:"listen" binding:"required,min=1,dive,required"` // 监听地址列表，如 [":80"]
	Protocols      []string                  `json:"protocols,omitempty"`                           // 允许的协议（如 h1、h2、h3）；省略使用 Caddy 默认值
	AutomaticHTTPS *ServerAutomaticHTTPSForm `json:"automatic_https,omitempty"`                     // 自动 HTTPS 设置；省略使用 Caddy 默认值
	StrictSNIHost  *bool                     `json:"strict_sni_host,omitempty"`                     // 是否严格校验 SNI 主机名
	IdleTimeout    string                    `json:"idle_timeout,omitempty"`                        // 空闲连接超时（Go duration 字符串）
	ReadTimeout    string                    `json:"read_timeout,omitempty"`                        // 读取请求超时（Go duration 字符串）
	WriteTimeout   string                    `json:"write_timeout,omitempty"`                       // 写入响应超时（Go duration 字符串）
	MaxHeaderBytes int                       `json:"max_header_bytes,omitempty"`                    // 请求头最大字节数
}

// ServerView 是不会暴露路由等子资源的服务视图。
type ServerView struct {
	ID   string `json:"id"`   // 服务视图标识（与名称相同）
	Name string `json:"name"` // Caddy server 名称
	ServerForm
	RouteCount int `json:"routeCount"` // 该服务下的路由数量
}

// ServerCreateRequest 创建服务的请求。
type ServerCreateRequest struct {
	Name string `json:"name" binding:"required"` // 新服务名称，全局唯一
	ServerForm
}

// ServerList 列出全部 HTTP server，结果按名称排序。
func (s *Service) ServerList(ctx context.Context) ([]ServerView, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Apps == nil || cfg.Apps.HTTP == nil || len(cfg.Apps.HTTP.Servers) == 0 {
		return []ServerView{}, nil
	}

	names := make([]string, 0, len(cfg.Apps.HTTP.Servers))
	for name := range cfg.Apps.HTTP.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]ServerView, 0, len(names))
	for _, name := range names {
		if srv := cfg.Apps.HTTP.Servers[name]; srv != nil {
			result = append(result, serverView(name, srv))
		}
	}
	return result, nil
}

// ServerInspect 获取指定 HTTP server。
func (s *Service) ServerInspect(ctx context.Context, ref string) (*ServerView, error) {
	name, err := serverNameFromRef(ref)
	if err != nil {
		return nil, err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, name)
	if srv == nil {
		return nil, ErrServerNotFound
	}
	view := serverView(name, srv)
	return &view, nil
}

// ServerCreate 创建 HTTP server。
func (s *Service) ServerCreate(ctx context.Context, req ServerCreateRequest) error {
	name, err := validateServerName(req.Name)
	if err != nil {
		return err
	}
	form, err := normalizeServerForm(req.ServerForm)
	if err != nil {
		return err
	}

	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		if getServer(cfg, name) != nil {
			return ErrServerExists
		}
		srv := &caddy.HTTPServer{}
		applyServerForm(srv, form)
		setServer(cfg, name, srv)
		return nil
	})
}

// ServerUpdate 更新指定 HTTP server，名称及未暴露的配置保持不变。
func (s *Service) ServerUpdate(ctx context.Context, ref string, req ServerForm) error {
	name, err := serverNameFromRef(ref)
	if err != nil {
		return err
	}
	form, err := normalizeServerForm(req)
	if err != nil {
		return err
	}

	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		srv := getServer(cfg, name)
		if srv == nil {
			return ErrServerNotFound
		}
		applyServerForm(srv, form)
		return nil
	})
}

// ServerDelete 删除指定 HTTP server 及其路由。
func (s *Service) ServerDelete(ctx context.Context, ref string) error {
	name, err := serverNameFromRef(ref)
	if err != nil {
		return err
	}

	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		if getServer(cfg, name) == nil {
			return ErrServerNotFound
		}
		if name == DefaultServerName {
			return ErrDefaultServer
		}
		if len(cfg.Apps.HTTP.Servers) <= 1 {
			return ErrLastServer
		}
		delete(cfg.Apps.HTTP.Servers, name)
		return nil
	})
}

// ─── 辅助函数 ───

func serverView(name string, srv *caddy.HTTPServer) ServerView {
	view := ServerView{
		ID:   encodeServerID(name),
		Name: name,
		ServerForm: ServerForm{
			Listen:         append([]string{}, srv.Listen...),
			Protocols:      append([]string(nil), srv.Protocols...),
			StrictSNIHost:  cloneBool(srv.StrictSNIHost),
			IdleTimeout:    srv.IdleTimeout.String(),
			ReadTimeout:    srv.ReadTimeout.String(),
			WriteTimeout:   srv.WriteTimeout.String(),
			MaxHeaderBytes: srv.MaxHeaderBytes,
		},
		RouteCount: len(srv.Routes),
	}
	if srv.AutomaticHTTPS != nil {
		view.AutomaticHTTPS = &ServerAutomaticHTTPSForm{
			Disable:          srv.AutomaticHTTPS.Disable,
			DisableRedirects: srv.AutomaticHTTPS.DisableRedirects,
		}
	}
	return view
}

func applyServerForm(srv *caddy.HTTPServer, form ServerForm) {
	// Config 的无损解码会在 Extras 中记住 omitempty 无法表达的显式空集合。
	// 这些字段属于本表单的完整 PUT 视图，更新时不能让旧的存在性标记覆盖新值。
	for _, key := range []string{"listen", "protocols", "automatic_https"} {
		delete(srv.Extras, key)
	}
	srv.Listen = append([]string(nil), form.Listen...)
	srv.Protocols = append([]string(nil), form.Protocols...)
	srv.StrictSNIHost = cloneBool(form.StrictSNIHost)
	srv.IdleTimeout = caddy.Duration(form.IdleTimeout)
	srv.ReadTimeout = caddy.Duration(form.ReadTimeout)
	srv.WriteTimeout = caddy.Duration(form.WriteTimeout)
	srv.MaxHeaderBytes = form.MaxHeaderBytes

	disable, disableRedirects := false, false
	if form.AutomaticHTTPS != nil {
		disable = form.AutomaticHTTPS.Disable
		disableRedirects = form.AutomaticHTTPS.DisableRedirects
	}
	if srv.AutomaticHTTPS != nil {
		srv.AutomaticHTTPS.Disable = disable
		srv.AutomaticHTTPS.DisableRedirects = disableRedirects
		if !disable && !disableRedirects && !hasUnmanagedAutomaticHTTPS(srv.AutomaticHTTPS) {
			srv.AutomaticHTTPS = nil
		}
	} else if disable || disableRedirects {
		srv.AutomaticHTTPS = &caddy.AutomaticHTTPS{
			Disable:          disable,
			DisableRedirects: disableRedirects,
		}
	}
}

func hasUnmanagedAutomaticHTTPS(config *caddy.AutomaticHTTPS) bool {
	clone := *config
	clone.Disable = false
	clone.DisableRedirects = false
	raw, err := json.Marshal(clone)
	return err != nil || string(raw) != "{}"
}

func setServer(cfg *caddy.Config, name string, srv *caddy.HTTPServer) {
	if cfg.Apps == nil {
		cfg.Apps = &caddy.AppsConfig{}
	}
	if cfg.Apps.HTTP == nil {
		cfg.Apps.HTTP = &caddy.HTTPApp{}
	}
	if cfg.Apps.HTTP.Servers == nil {
		cfg.Apps.HTTP.Servers = map[string]*caddy.HTTPServer{}
	}
	cfg.Apps.HTTP.Servers[name] = srv
}

func normalizeServerForm(form ServerForm) (ServerForm, error) {
	if len(form.Listen) == 0 {
		return ServerForm{}, fmt.Errorf("%w: 至少需要一个监听地址", ErrServerConfigInvalid)
	}
	listenSeen := make(map[string]struct{}, len(form.Listen))
	for i, listen := range form.Listen {
		listen = strings.TrimSpace(listen)
		if listen == "" {
			return ServerForm{}, fmt.Errorf("%w: 监听地址不能为空", ErrServerConfigInvalid)
		}
		if _, exists := listenSeen[listen]; exists {
			return ServerForm{}, fmt.Errorf("%w: 监听地址 %q 重复", ErrServerConfigInvalid, listen)
		}
		listenSeen[listen] = struct{}{}
		form.Listen[i] = listen
	}

	protocolSeen := make(map[string]struct{}, len(form.Protocols))
	for i, protocol := range form.Protocols {
		protocol = strings.TrimSpace(protocol)
		switch protocol {
		case "h1", "h2", "h2c", "h3":
		default:
			return ServerForm{}, fmt.Errorf("%w: 不支持协议 %q", ErrServerConfigInvalid, protocol)
		}
		if _, exists := protocolSeen[protocol]; exists {
			return ServerForm{}, fmt.Errorf("%w: 协议 %q 重复", ErrServerConfigInvalid, protocol)
		}
		protocolSeen[protocol] = struct{}{}
		form.Protocols[i] = protocol
	}
	if _, h1 := protocolSeen["h1"]; !h1 {
		if _, h2 := protocolSeen["h2"]; h2 {
			return ServerForm{}, fmt.Errorf("%w: h2 依赖 h1", ErrServerConfigInvalid)
		}
		if _, h2c := protocolSeen["h2c"]; h2c {
			return ServerForm{}, fmt.Errorf("%w: h2c 依赖 h1", ErrServerConfigInvalid)
		}
	}

	if form.MaxHeaderBytes < 0 {
		return ServerForm{}, fmt.Errorf("%w: max_header_bytes 不能为负数", ErrServerConfigInvalid)
	}
	return form, nil
}

func validateServerName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%w: 名称不能为空", ErrServerNameInvalid)
	}
	if !utf8.ValidString(name) {
		return "", fmt.Errorf("%w: 名称必须是有效的 UTF-8", ErrServerNameInvalid)
	}
	return name, nil
}

func encodeServerID(name string) string {
	return serverIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(name))
}

func serverNameFromRef(ref string) (string, error) {
	if ref == "" {
		return DefaultServerName, nil
	}
	name := ref
	if strings.HasPrefix(ref, serverIDPrefix) {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ref, serverIDPrefix))
		if err != nil {
			return "", fmt.Errorf("%w: id 无法解码", ErrServerNameInvalid)
		}
		name = string(raw)
	}
	return validateServerName(name)
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
