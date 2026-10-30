package gateway

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/rehiy/libgo/websocket"

	"isrvd/pkgs/helper"
)

// nodePath 处理 /n/<节点ID>/ 下的全部请求：去掉前缀后，页面与中控专属模块交给中控，
// 业务模块经认证、权限与审计后转发到节点。
func (g *gateway) nodePath(w http.ResponseWriter, r *http.Request) {
	id, rest := r.PathValue("id"), r.PathValue("rest")
	if !nodeIDPattern.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	// 去掉 /n/<节点ID> 前缀后，其余处理与直接访问中控一致
	r.URL.Path = "/" + rest
	r.URL.RawPath = ""

	if !strings.HasPrefix(rest, "api/") {
		g.proxy.ServeHTTP(w, r) // 页面与静态资源
		return
	}
	module, _, _ := strings.Cut(strings.TrimPrefix(rest, "api/"), "/")
	switch {
	case module == "node":
		// 节点管理属于中控：在节点视角下打开的页面同样要能管理节点，按中控自己的接口处理
		g.mux.ServeHTTP(w, r)
	case !g.nodeModules[module]:
		g.proxy.ServeHTTP(w, r) // 账号、AI 助手、SSH 等始终作用于中控
	case rest == "api/overview/bootstrap":
		g.nodeBootstrap(w, r, id)
	default:
		g.authorized(g.nodeForward)(w, r)
	}
}

// nodeForward 在 authorized 的认证、权限校验和审计下把请求经隧道转发到节点。
// 受管机内部仍使用本地服务令牌。
func (g *gateway) nodeForward(w http.ResponseWriter, r *http.Request) {
	// 与 isrvd 本机终端一致，WebSocket 必须校验 Origin；受管机转发时会去掉 Origin，所以只能在这里把关
	if helper.IsWebSocket(r) && !g.originAllowed(r) {
		fail(w, r, http.StatusForbidden, "不允许的跨站来源")
		return
	}
	g.Nodes.ProxyServe(w, g.withoutCredentials(r), r.PathValue("id"))
}

// withoutCredentials 返回去掉用户凭据的请求副本：用户凭据只属于中控，不下发给节点。
// 用副本而不改原请求，审计仍能看到完整的原始请求。
func (g *gateway) withoutCredentials(r *http.Request) *http.Request {
	out := r.Clone(r.Context())
	for _, name := range g.CredentialHeaders() {
		out.Header.Del(name)
	}
	if q := out.URL.Query(); q.Has("token") {
		q.Del("token")
		out.URL.RawQuery = q.Encode()
	}
	return out
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
	if g.AllowedOrigins == nil {
		return false
	}
	allowed := g.AllowedOrigins()
	return len(allowed) > 0 && (&websocket.ServerConfig{AllowedOrigins: allowed}).CheckOrigin(origin)
}

// nodeIDPattern 节点 ID 为 10 位以上的字母、数字或横线。
var nodeIDPattern = regexp.MustCompile(`^[0-9A-Za-z-]{10,}$`)
