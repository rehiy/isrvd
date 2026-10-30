package gateway

import (
	"bufio"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"isrvd/server/config"
	"isrvd/server/service/account"
)

// Auditor 把网关处理的操作记入 isrvd 的审计日志（实现见 app.Audit）
type Auditor interface {
	BodyRead(r *http.Request) string
	AuditRecord(r *http.Request, username, ip string, statusCode int, startTime time.Time, body string)
}

// trustIdentity 在入口处清理并重建网关身份头：外部传入的一律丢弃，
// 启用代理 Header 认证时由网关自己校验，并把结果写入内部身份头，原始认证头不再透传。
func trustIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := config.Current()
		r.Header.Del(account.GatewayUserHeader)
		if snapshot.THA.Enabled {
			user, _ := account.HeaderAuth(snapshot, r)
			r.Header.Del(snapshot.THA.HeaderName)
			if user != "" {
				r.Header.Set(account.GatewayUserHeader, user)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// authorized 统一执行节点管理与业务转发的认证、路由权限校验和审计。
func (g *gateway) authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, status, err := g.authorize(r, g.allowQueryToken(r))
		if err != nil {
			fail(w, r, status, err.Error())
			return
		}
		g.audited(w, r, user, h)
	}
}

// audited 执行 h，并按 isrvd 的审计基线（非 GET 与全部 WebSocket）把操作记入审计日志。
// 与 app 的审计中间件一致：认证失败的请求不记录，WebSocket 在会话结束时记录。
func (g *gateway) audited(w http.ResponseWriter, r *http.Request, user string, h http.HandlerFunc) {
	ws := isWebSocket(r)
	if g.audit == nil || (r.Method == http.MethodGet && !ws) {
		h(w, r)
		return
	}

	start := time.Now()
	body := ""
	if !ws {
		body = g.audit.BodyRead(r)
	}
	rec := &statusRecorder{ResponseWriter: w}
	h(rec, r)
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	g.audit.AuditRecord(r, user, clientIP(r), status, start, body)
}

// allowQueryToken 与 isrvd 一样，根据 Upgrade 头或声明了 QueryToken 的 GET 路由读取 ?token=。
// 这里不校验 WebSocket 握手；凭据与路由权限仍由中控统一验证。
func (g *gateway) allowQueryToken(r *http.Request) bool {
	if isWebSocket(r) {
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

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// clientIP 只取连接对端 IP，不信任客户端提供的转发头。
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// statusRecorder 记录响应状态码；Hijack（WebSocket 升级）由它接管以记录 101，
// 其余能力（如 Flush）经 Unwrap 交给 http.ResponseController 透传。
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

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(s.ResponseWriter).Hijack()
	if err == nil && s.status == 0 {
		// WebSocket 握手劫持连接后直接写入 101，不再经过 WriteHeader。
		s.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
