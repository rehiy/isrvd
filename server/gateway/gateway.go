package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/config"
	"isrvd/server/i18n"
	"isrvd/server/service/account"
	"isrvd/server/service/node"
)

// Auditor 把网关处理的操作记入 isrvd 的审计日志（实现见 app.Audit）
type Auditor interface {
	BodyRead(r *http.Request) string
	AuditRecord(r *http.Request, username, ip string, statusCode int, startTime time.Time, body string)
}

// Options 中控网关的装配参数
type Options struct {
	Upstream         *url.URL        // 同进程 isrvd 的回环地址
	Nodes            *node.Service   // 节点服务
	AllowedOrigins   func() []string // isrvd 配置的跨域白名单，用于校验转发到节点的 WebSocket；每次校验时读取，配置重载后即时生效
	QueryTokenRoutes []string        // 允许 ?token= 认证的 GET 路由，来自 app.QueryTokenRoutes
	Audit            Auditor         // 审计入口；节点管理与节点上的写操作、终端会话按 isrvd 的审计基线记录

	// Authenticate 复用应用当前账户服务；nil 成员表示未登录，error 表示服务不可用。
	Authenticate func(*http.Request, bool) (*account.MemberInfo, error)

	// CheckPermission 复用中控当前路由与账户权限；返回 HTTP 状态码及错误。
	CheckPermission func(*http.Request, string) (int, error)
}

// Handler 返回中控网关：节点管理接口与 /n/<节点ID>/ 下的节点转发由它处理，
// 其余请求（含全部页面）原样转发给 upstream（同进程的 isrvd，只监听回环地址）。
// 节点切换与节点管理页面由 webview 实现，通过 isrvd 提供。
func Handler(opt Options) http.Handler {
	return newGateway(opt).routes()
}

type gateway struct {
	nodes    *node.Service
	upstream *url.URL
	proxy    *httputil.ReverseProxy // 转发到中控自己的 isrvd
	client   *http.Client           // 仅用于页面 bootstrap 聚合

	allowedOrigins  func() []string // 额外允许的 WebSocket Origin（isrvd 的 allowedOrigins），默认仅同源
	audit           Auditor
	authenticate    func(*http.Request, bool) (*account.MemberInfo, error)
	checkPermission func(*http.Request, string) (int, error)

	connect http.Handler   // 受管机建立隧道的入口
	mux     *http.ServeMux // 对外路由表；节点视角下的 /api/node/* 也经它分发

	queryRoutes [][]string // 允许 ?token= 的 GET 路由，按路径段拆分
}

func newGateway(opt Options) *gateway {
	upstream := opt.Upstream
	g := &gateway{
		nodes:           opt.Nodes,
		upstream:        upstream,
		allowedOrigins:  opt.AllowedOrigins,
		audit:           opt.Audit,
		authenticate:    opt.Authenticate,
		checkPermission: opt.CheckPermission,
		client: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	for _, route := range opt.QueryTokenRoutes {
		g.queryRoutes = append(g.queryRoutes, strings.Split(route, "/"))
	}
	g.connect = g.nodeConnect()
	g.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(upstream)
			pr.SetXForwarded()
			if user := pr.In.Header.Get(account.GatewayUserHeader); user != "" {
				pr.Out.Header.Set(account.GatewayUserHeader, user)
			}
		},
		FlushInterval: -1, // SSE、日志流与 WebSocket 需要立即刷新
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			logman.Warn("转发到中控 isrvd 失败", "path", r.URL.Path, "error", err)
			fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		},
	}
	return g
}

func (g *gateway) routes() http.Handler {
	mux := http.NewServeMux()

	// 节点管理：路由与可分配权限共用 service/node 的元信息。
	handlers := map[string]func(http.ResponseWriter, *http.Request, string){
		"GET /api/node/nodes":            g.nodeList,
		"PUT /api/node/item/:id":         g.nodeUpdate,
		"POST /api/node/item/:id/revoke": g.nodeRevoke,
		"DELETE /api/node/item/:id":      g.nodeDelete,
		"GET /api/node/codes":            g.nodeCodeList,
		"POST /api/node/code":            g.nodeCodeCreate,
		"DELETE /api/node/code/:id":      g.nodeCodeDelete,
	}
	for _, route := range node.PermissionRoutes() {
		key := route.Method + " " + route.Path
		mux.HandleFunc(strings.ReplaceAll(key, ":id", "{id}"), g.permissionOnly(handlers[key]))
	}

	// 受管机接入：凭节点令牌连接，首次连接时待接入的令牌自动激活为节点
	mux.Handle("GET /api/node/connect", g.connect)

	// 以某个节点的视角访问：/n/<节点ID>/ 提供页面，/n/<节点ID>/api/ 转发业务接口
	mux.HandleFunc("/n/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/n/"+r.PathValue("id")+"/", http.StatusFound)
	})
	mux.HandleFunc("/n/{id}/{rest...}", g.nodePath)

	mux.Handle("/", g.proxy)
	g.mux = mux
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := config.Current()
		r = r.Clone(r.Context())
		r.Header.Del(account.GatewayUserHeader)
		if snapshot.THA.Enabled {
			user, _ := account.HeaderAuth(snapshot, r)
			r.Header.Del(snapshot.THA.HeaderName)
			if user != "" {
				r.Header.Set(account.GatewayUserHeader, user)
			}
		}
		mux.ServeHTTP(w, r)
	})
}

// ─── 节点视角 ───

// nodeModules 可按节点转发的业务模块。其余模块（账号、系统、AI 助手、SSH 等）
// 即使在 /n/<节点ID>/ 下访问，也始终由中控自己的 isrvd 处理。
var nodeModules = map[string]bool{
	"overview": true, "filer": true, "shell": true, "local": true,
	"docker": true, "swarm": true, "compose": true, "cron": true,
	"apisix": true, "caddy": true,
}

func (g *gateway) nodePath(w http.ResponseWriter, r *http.Request) {
	id, rest := r.PathValue("id"), r.PathValue("rest")
	if !validNodeID(id) {
		http.NotFound(w, r)
		return
	}
	// 去掉 /n/<节点ID> 前缀后，其余处理与直接访问中控一致
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/" + rest
	r2.URL.RawPath = ""

	if !strings.HasPrefix(rest, "api/") {
		g.proxy.ServeHTTP(w, r2) // 页面与静态资源
		return
	}
	module, _, _ := strings.Cut(strings.TrimPrefix(rest, "api/"), "/")
	switch {
	case module == "node":
		// 节点管理属于中控：在节点视角下打开的页面同样要能管理节点，按中控自己的接口处理
		g.mux.ServeHTTP(w, r2)
	case !nodeModules[module]:
		g.proxy.ServeHTTP(w, r2) // 账号、系统、AI 助手、SSH 等始终作用于中控
	case rest == "api/overview/bootstrap":
		g.nodeBootstrap(w, r2, id)
	default:
		g.nodeForward(w, r2, id)
	}
}

// nodeForward 校验调用者后把请求经隧道转发到节点。
// 转发前按中控的业务路由权限校验；受管机内部仍使用本地服务令牌。
func (g *gateway) nodeForward(w http.ResponseWriter, r *http.Request, id string) {
	member, ok := g.authorize(w, r)
	if !ok {
		return
	}
	g.audited(w, r, member.Username, func(w http.ResponseWriter, r *http.Request) {
		// 与 isrvd 本机终端一致，WebSocket 必须校验 Origin；受管机转发时会去掉 Origin，所以只能在这里把关
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") && !g.originAllowed(r) {
			fail(w, r, http.StatusForbidden, "不允许的跨站来源")
			return
		}
		g.nodes.ProxyServe(w, r, id)
	})
}

// originAllowed 判断 WebSocket 的 Origin 是否可接受：无 Origin（非浏览器客户端，仍需有效令牌）、
// 与当前访问同源、或匹配 isrvd 配置的 allowedOrigins（支持通配符）。
func (g *gateway) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if u, err := url.Parse(origin); err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host) {
		return true
	}
	// 未配置白名单时 CheckOrigin 会放行一切，这里要求跨站来源必须被明确列出
	if g.allowedOrigins == nil {
		return false
	}
	allowed := g.allowedOrigins()
	return len(allowed) > 0 && (&websocket.ServerConfig{AllowedOrigins: allowed}).CheckOrigin(origin)
}

// nodeBootstrap 以中控的认证信息、节点自身的服务可用性组合启动数据，
// 让页面在所选节点下按该节点实际可用的服务显示菜单。
func (g *gateway) nodeBootstrap(w http.ResponseWriter, r *http.Request, id string) {
	status, body, err := g.bootstrapCall(r)
	if err != nil {
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return
	}
	if status != http.StatusOK {
		writeRaw(w, status, body)
		return
	}

	var env map[string]any
	if json.Unmarshal(body, &env) != nil {
		writeRaw(w, status, body)
		return
	}

	payload, _ := env["payload"].(map[string]any)
	if payload == nil {
		writeRaw(w, status, body)
		return
	}

	// 使用 bootstrap 可选认证返回的身份组合节点启动数据。
	auth, _ := payload["auth"].(map[string]any)
	username, _ := auth["username"].(string)
	if username == "" {
		writeRaw(w, status, body)
		return
	}

	// 节点管理（node）与 AI 助手（copilot）由中控提供，保留中控的结果；
	// 其余服务以节点为准，节点不可达时全部视为不可用
	probe := map[string]any{"node": false, "copilot": false, "apisix": false, "caddy": false, "docker": false, "swarm": false, "compose": false}
	if local, ok := payload["probe"].(map[string]any); ok {
		for _, k := range []string{"node", "copilot"} {
			if v, ok := local[k]; ok {
				probe[k] = v
			}
		}
	}
	if data, err := g.nodes.Fetch(r.Context(), id, "/api/overview/bootstrap"); err == nil {
		var remote struct {
			Payload struct {
				Probe map[string]any `json:"probe"`
			} `json:"payload"`
		}
		if json.Unmarshal(data, &remote) == nil {
			for _, k := range []string{"apisix", "caddy", "docker", "swarm", "compose"} {
				if v, ok := remote.Payload.Probe[k]; ok {
					probe[k] = v
				}
			}
		}
	}
	payload["probe"] = probe

	out, err := json.Marshal(env)
	if err != nil {
		writeRaw(w, status, body)
		return
	}
	writeRaw(w, http.StatusOK, out)
}

// ─── 身份 ───

// permissionOnly 认证并校验节点管理路由权限
func (g *gateway) permissionOnly(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		member, ok := g.authorize(w, r)
		if !ok {
			return
		}
		g.audited(w, r, member.Username, func(w http.ResponseWriter, r *http.Request) {
			h(w, r, member.Username)
		})
	}
}

// audited 执行 h，并按 isrvd 的审计基线（非 GET 与全部 WebSocket）把操作记入审计日志。
// 与 app 的审计中间件一致：认证失败的请求不记录，WebSocket 在会话结束时记录。
func (g *gateway) audited(w http.ResponseWriter, r *http.Request, user string, h func(http.ResponseWriter, *http.Request)) {
	isWS := strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
	if g.audit == nil || (r.Method == http.MethodGet && !isWS) {
		h(w, r)
		return
	}

	start := time.Now()
	body := ""
	if !isWS {
		body = g.audit.BodyRead(r)
	}
	rec := &statusRecorder{ResponseWriter: w}
	h(rec, r)
	g.audit.AuditRecord(r, user, g.clientIP(r), rec.status, start, body)
}

// authorize 认证调用者并检查请求路由权限；不满足时已写出响应。
// 中控 isrvd 不可用与令牌无效必须区分：前者返回 502，否则前端会把一次上游抖动当成登录过期而清掉登录态。
func (g *gateway) authorize(w http.ResponseWriter, r *http.Request) (*account.MemberInfo, bool) {
	if g.authenticate == nil {
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return nil, false
	}
	member, err := g.authenticate(r, g.allowQueryToken(r))
	switch {
	case err != nil:
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return nil, false
	case member == nil:
		fail(w, r, http.StatusUnauthorized, "未登录或登录已过期")
		return nil, false
	}
	if g.checkPermission == nil {
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return nil, false
	}
	if status, err := g.checkPermission(r, member.Username); err != nil {
		fail(w, r, status, err.Error())
		return nil, false
	}
	return member, true
}

// bootstrapCall 仅为节点视角的页面加载中控启动数据，不用于业务请求的身份校验。
// 只传递认证与语言信息，避免把浏览器的条件请求、升级或内容头带入聚合查询。
func (g *gateway) bootstrapCall(r *http.Request) (int, []byte, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, g.upstream.JoinPath("api/overview/bootstrap").String(), nil)
	if err != nil {
		return 0, nil, err
	}
	for _, name := range []string{"Authorization", "Accept-Language", account.GatewayUserHeader} {
		if value := r.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}
	if lang := r.URL.Query().Get("lang"); lang != "" {
		req.URL.RawQuery = url.Values{"lang": {lang}}.Encode()
	}
	// WebSocket 与下载无法携带请求头，凭据在 ?token= 中；仅限 isrvd 允许的范围
	if req.Header.Get("Authorization") == "" && g.allowQueryToken(r) {
		if token := r.URL.Query().Get("token"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, body, err
}

// allowQueryToken 与 isrvd 一样，根据 Upgrade 头或声明了 QueryToken 的 GET 路由读取 ?token=。
// 这里不校验 WebSocket 握手；凭据是否有效及是否为创始人仍由中控统一验证。
func (g *gateway) allowQueryToken(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}
	if r.Method != http.MethodGet {
		return false
	}
	segs := strings.Split(r.URL.Path, "/")
	return slices.ContainsFunc(g.queryRoutes, func(pattern []string) bool {
		return len(pattern) == len(segs) && slices.EqualFunc(pattern, segs, func(p, s string) bool {
			return p == s || (strings.HasPrefix(p, ":") && s != "")
		})
	})
}

// ─── 辅助函数 ───

// clientIP 只取连接对端 IP，不信任客户端提供的转发头。
func (g *gateway) clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// validNodeID 节点 ID 为 strutil.NewString 生成的 UUID（小写十六进制，8-4-4-4-12）
func validNodeID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case (c < '0' || c > '9') && (c < 'a' || c > 'f'):
			return false
		}
	}
	return true
}

// statusRecorder 记录响应状态码，同时保持 Flush、Hijack（WebSocket 升级）可用
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() { _ = http.NewResponseController(s.ResponseWriter).Flush() }

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(s.ResponseWriter).Hijack()
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func respond(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func success(w http.ResponseWriter, r *http.Request, message string, payload any) {
	// 与 fail 一致：按请求语言翻译提示文案
	respond(w, http.StatusOK, map[string]any{
		"success": true,
		"message": i18n.Translate(i18n.Parse(r.Header.Get("Accept-Language")), message),
		"payload": payload,
	})
}

func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	// 网关是原生 http.Handler，没有 gin 上下文，按请求头直接协商语言
	lang := i18n.Parse(r.Header.Get("Accept-Language"))
	respond(w, status, map[string]any{"success": false, "message": i18n.Translate(lang, message)})
}
