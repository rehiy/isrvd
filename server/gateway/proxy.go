package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rehiy/libgo/websocket"

	"isrvd/server/service/account"
)

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
		g.permissionOnly(g.nodeForward)(w, r2)
	}
}

// nodeForward 在 permissionOnly 的认证、权限校验和审计下把请求经隧道转发到节点。
// 受管机内部仍使用本地服务令牌。
func (g *gateway) nodeForward(w http.ResponseWriter, r *http.Request) {
	// 与 isrvd 本机终端一致，WebSocket 必须校验 Origin；受管机转发时会去掉 Origin，所以只能在这里把关
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") && !g.originAllowed(r) {
		fail(w, r, http.StatusForbidden, "不允许的跨站来源")
		return
	}
	g.nodes.ProxyServe(w, r, r.PathValue("id"))
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

// ─── 辅助函数 ───

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
