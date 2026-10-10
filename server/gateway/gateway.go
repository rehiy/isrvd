package gateway

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/ttlcache"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/i18n"
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
	TrustProxy       bool            // 前面还有可信反向代理时按 X-Forwarded-For 识别来源 IP
	AllowedOrigins   func() []string // isrvd 配置的跨域白名单，用于校验转发到节点的 WebSocket；每次校验时读取，配置重载后即时生效
	QueryTokenRoutes []string        // 允许 ?token= 认证的 GET 路由，来自 app.QueryTokenRoutes
	Audit            Auditor         // 审计入口；节点管理与节点上的写操作、终端会话按 isrvd 的审计基线记录
}

// Handler 返回中控网关：节点管理接口与 /n/<节点ID>/ 下的节点转发由它处理，
// 其余请求（含全部页面）原样转发给 upstream（同进程的 isrvd，只监听回环地址）。
// 节点切换与节点管理页面是 webview 的一部分，由 isrvd 提供，网关不再注入或托管任何页面。
func Handler(opt Options) http.Handler {
	return newGateway(opt).routes()
}

// nodeModules 可按节点转发的业务模块。其余模块（账号、系统、AI 助手、SSH 等）
// 即使在 /n/<节点ID>/ 下访问，也始终由中控自己的 isrvd 处理。
var nodeModules = map[string]bool{
	"overview": true, "filer": true, "shell": true, "local": true,
	"docker": true, "swarm": true, "compose": true, "cron": true,
	"apisix": true, "caddy": true,
}

const identityTTL = 10 * time.Second

// identity 经中控 isrvd 确认的调用者身份
type identity struct {
	user    string
	founder bool
}

type gateway struct {
	nodes      *node.Service
	upstream   *url.URL
	proxy      *httputil.ReverseProxy // 转发到中控自己的 isrvd
	client     *http.Client           // 向中控 isrvd 查询身份
	trustProxy bool

	allowedOrigins func() []string // 额外允许的 WebSocket Origin（isrvd 的 allowedOrigins），默认仅同源
	audit          Auditor

	cache   *ttlcache.TimedCache // 令牌哈希 → identity，缓存 identityTTL
	connect http.Handler         // 受管机建立隧道的入口
	mux     *http.ServeMux       // 对外路由表；节点视角下的 /api/node/* 也经它分发

	queryRoutes [][]string // 允许 ?token= 的 GET 路由，按路径段拆分
}

func newGateway(opt Options) *gateway {
	upstream := opt.Upstream
	g := &gateway{
		nodes:          opt.Nodes,
		upstream:       upstream,
		trustProxy:     opt.TrustProxy,
		allowedOrigins: opt.AllowedOrigins,
		audit:          opt.Audit,
		cache:          ttlcache.NewTimedCache(identityTTL),
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
			// 保留链路上已有的 X-Forwarded-For，让 isrvd 的审计记录真实来源
			pr.Out.Header["X-Forwarded-For"] = pr.In.Header["X-Forwarded-For"]
			pr.SetURL(upstream)
			pr.SetXForwarded()
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

	// 节点管理（仅创始人）
	mux.HandleFunc("GET /api/node/nodes", g.founderOnly(g.nodeList))
	mux.HandleFunc("PUT /api/node/item/{id}", g.founderOnly(g.nodeUpdate))
	mux.HandleFunc("POST /api/node/item/{id}/approve", g.founderOnly(g.nodeApprove))
	mux.HandleFunc("POST /api/node/item/{id}/revoke", g.founderOnly(g.nodeRevoke))
	mux.HandleFunc("DELETE /api/node/item/{id}", g.founderOnly(g.nodeDelete))
	mux.HandleFunc("GET /api/node/codes", g.founderOnly(g.nodeCodeList))
	mux.HandleFunc("POST /api/node/code", g.founderOnly(g.nodeCodeCreate))
	mux.HandleFunc("DELETE /api/node/code/{id}", g.founderOnly(g.nodeCodeDelete))

	// 受管机接入：注册与领取凭一次性凭据，连接凭节点令牌
	mux.HandleFunc("POST /api/node/enroll", g.nodeEnroll)
	mux.HandleFunc("POST /api/node/enroll/claim", g.nodeClaim)
	mux.Handle("GET /api/node/connect", g.connect)

	// 以某个节点的视角访问：/n/<节点ID>/ 提供页面，/n/<节点ID>/api/ 转发业务接口
	mux.HandleFunc("/n/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/n/"+r.PathValue("id")+"/", http.StatusFound)
	})
	mux.HandleFunc("/n/{id}/{rest...}", g.nodePath)

	mux.Handle("/", g.proxy)
	g.mux = mux
	return mux
}

// ─── 节点视角 ───

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
// 只有创始人可以操作节点：节点上的请求以受管机本地签发的创始人令牌身份执行，不区分调用者，
// 因此不能把节点权限下放给只有部分路由权限的成员。
func (g *gateway) nodeForward(w http.ResponseWriter, r *http.Request, id string) {
	ident, ok := g.authorize(w, r)
	if !ok {
		return
	}
	g.audited(w, r, ident.user, func(w http.ResponseWriter, r *http.Request) {
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
	ident := parseIdentity(body)
	if status != http.StatusOK || ident.user == "" || !ident.founder {
		writeRaw(w, status, body) // 未登录或非创始人：原样返回，由后续业务请求给出明确错误
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

// founderOnly 要求调用者是中控 isrvd 的创始人
func (g *gateway) founderOnly(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ident, ok := g.authorize(w, r)
		if !ok {
			return
		}
		g.audited(w, r, ident.user, func(w http.ResponseWriter, r *http.Request) {
			h(w, r, ident.user)
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

// authorize 要求调用者是创始人；不满足时已写出响应。
// 中控 isrvd 不可用与令牌无效必须区分：前者返回 502，否则前端会把一次上游抖动当成登录过期而清掉登录态。
func (g *gateway) authorize(w http.ResponseWriter, r *http.Request) (identity, bool) {
	ident, err := g.identify(r)
	switch {
	case err != nil:
		fail(w, r, http.StatusBadGateway, "中控 isrvd 不可用")
		return ident, false
	case ident.user == "":
		fail(w, r, http.StatusUnauthorized, "未登录或登录已过期")
		return ident, false
	case !ident.founder:
		fail(w, r, http.StatusForbidden, "仅创始人可管理和操作节点")
		return ident, false
	}
	return ident, true
}

// identify 把调用者的凭据交给中控 isrvd 验证并取回身份；err 表示上游不可用，而不是凭据无效。
// 带令牌的结果缓存 identityTTL，避免每个请求都触发一次启动聚合（含服务探活）。
func (g *gateway) identify(r *http.Request) (identity, error) {
	key := ""
	if token := g.credential(r); token != "" {
		sum := sha256.Sum256([]byte(token))
		key = hex.EncodeToString(sum[:])
		if v, ok := g.cache.Get(key); ok {
			return v.(identity), nil
		}
	}

	status, body, err := g.bootstrapCall(r)
	if err != nil {
		return identity{}, err
	}
	if status != http.StatusOK {
		return identity{}, fmt.Errorf("中控 isrvd 返回 HTTP %d", status)
	}
	ident := parseIdentity(body)
	if key != "" && ident.user != "" {
		g.cache.Set(key, ident)
	}
	return ident, nil
}

// skipHeaders 向中控 isrvd 查询身份时不转发的请求头
var skipHeaders = map[string]bool{
	"Accept": true, "Accept-Encoding": true, "Connection": true, "Content-Length": true,
	"Content-Type": true, "Keep-Alive": true, "Origin": true, "Proxy-Connection": true,
	"Range": true, "Te": true, "Transfer-Encoding": true, "Upgrade": true,
	"If-Match": true, "If-None-Match": true, "If-Modified-Since": true, "If-Range": true,
}

// bootstrapCall 以调用者的凭据请求中控 isrvd 的启动聚合接口。
// 该接口允许匿名访问：凭据无效时返回成功但不含用户名。
func (g *gateway) bootstrapCall(r *http.Request) (int, []byte, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, g.upstream.JoinPath("api/overview/bootstrap").String(), nil)
	if err != nil {
		return 0, nil, err
	}
	for k, vs := range r.Header {
		if skipHeaders[http.CanonicalHeaderKey(k)] || strings.HasPrefix(http.CanonicalHeaderKey(k), "Sec-Websocket-") {
			continue
		}
		req.Header[k] = slices.Clone(vs)
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

func parseIdentity(body []byte) identity {
	var env struct {
		Payload struct {
			Auth struct {
				Username string `json:"username"`
				Member   *struct {
					Founder bool `json:"founder"`
				} `json:"member"`
			} `json:"auth"`
		} `json:"payload"`
	}
	if json.Unmarshal(body, &env) != nil || env.Payload.Auth.Username == "" || env.Payload.Auth.Member == nil {
		return identity{}
	}
	return identity{user: env.Payload.Auth.Username, founder: env.Payload.Auth.Member.Founder}
}

// credential 取请求携带的令牌：Authorization 头优先，其次（仅在允许时）?token=
func (g *gateway) credential(r *http.Request) string {
	if v := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")); v != "" {
		return v
	}
	if g.allowQueryToken(r) {
		return r.URL.Query().Get("token")
	}
	return ""
}

// allowQueryToken 判断该请求能否用 ?token= 认证，规则与 isrvd 一致：
// WebSocket 升级，或 isrvd 声明了 QueryToken 的 GET 路由（SSE、文件下载）。
// 其余请求（尤其是写操作与节点管理接口）必须用 Authorization 头，避免令牌进入访问日志、历史记录与 Referer。
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

// clientIP 返回请求来源 IP，仅在明确声明前面有可信代理时才采信 X-Forwarded-For
func (g *gateway) clientIP(r *http.Request) string {
	if g.trustProxy {
		if first, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ","); strings.TrimSpace(first) != "" {
			return strings.TrimSpace(first)
		}
	}
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
