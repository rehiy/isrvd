package gateway

import (
	"bufio"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"isrvd/server/service/account"
)

// Auditor 把网关处理的操作记入 isrvd 的审计日志（实现见 app.Audit）
type Auditor interface {
	BodyRead(r *http.Request) string
	AuditRecord(r *http.Request, username, ip string, statusCode int, startTime time.Time, body string)
}

// permissionOnly 统一执行节点管理与业务转发的认证、权限校验和审计。
func (g *gateway) permissionOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		member, ok := g.authorize(w, r)
		if !ok {
			return
		}
		g.audited(w, r, member.Username, h)
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
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	g.audit.AuditRecord(r, user, g.clientIP(r), status, start, body)
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

// allowQueryToken 与 isrvd 一样，根据 Upgrade 头或声明了 QueryToken 的 GET 路由读取 ?token=。
// 这里不校验 WebSocket 握手；凭据与路由权限仍由中控统一验证。
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

// clientIP 只取连接对端 IP，不信任客户端提供的转发头。
func (g *gateway) clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// statusRecorder 记录响应状态码，同时保持 Flush、Hijack（WebSocket 升级）可用
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status != 0 {
		return
	}
	// 1xx 临时响应不会决定最终状态，101 则是 WebSocket 升级的最终响应。
	if code < 100 || code >= 200 || code == http.StatusSwitchingProtocols {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() {
	if s.status == 0 {
		s.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(s.ResponseWriter).Flush()
}

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(s.ResponseWriter).Hijack()
	if err == nil && s.status == 0 {
		// WebSocket 握手劫持连接后直接写入 101，不再经过 WriteHeader。
		s.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
