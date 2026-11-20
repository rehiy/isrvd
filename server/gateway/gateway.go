package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/helper"
	"isrvd/server/service/node"
)

// Options 中控网关的装配参数
type Options struct {
	Upstream         *url.URL        // 同进程 isrvd 的回环地址
	Nodes            *node.Service   // 节点服务
	AllowedOrigins   func() []string // isrvd 配置的跨域白名单，用于校验转发到节点的 WebSocket；每次校验时读取，配置重载后即时生效
	QueryTokenRoutes []string        // 允许 ?token= 认证的 GET 路由，来自 app.GatewayRoutes
	NodeModules      []string        // 节点视角下转发给受管机的业务模块，来自 app.GatewayRoutes
	UserHeader       string          // 内部身份头名称；转发到中控自己的 isrvd 时原样带上
	Audit            Auditor         // 审计入口；节点管理与节点上的写操作、终端会话按 isrvd 的审计基线记录

	// TrustIdentity 在入口处清理并重建内部身份头：外部传入的一律丢弃；
	// 启用代理 Header 认证时由实现校验来源，把结果写入内部身份头，原始认证头不再透传。
	TrustIdentity func(r *http.Request)
	// CredentialHeaders 返回携带用户凭据、不得下发给受管机的请求头（含内部身份头）。
	CredentialHeaders func() []string

	// Authorize 复用应用当前账户服务认证调用者并校验路由权限，返回用户名；
	// 失败时返回应写出的 HTTP 状态码及错误。
	Authorize func(r *http.Request, allowQueryToken bool) (string, int, error)
}

type gateway struct {
	Options

	proxy  *httputil.ReverseProxy // 转发到中控自己的 isrvd
	client *http.Client           // 仅用于页面 bootstrap 聚合
	mux    *http.ServeMux         // 对外路由表；节点视角下的 /api/node/* 也经它分发

	queryRoutes [][]string      // 允许 ?token= 的 GET 路由，按路径段拆分
	nodeModules map[string]bool // 节点视角下转发给受管机的业务模块
}

// Handler 返回中控网关：节点管理接口与 /n/<节点ID>/ 下的节点转发由它处理，
// 其余请求（含全部页面）原样转发给 upstream（同进程的 isrvd，只监听回环地址）。
// 节点切换与节点管理页面由 webview 实现，通过 isrvd 提供。
func Handler(opt Options) http.Handler {
	g := &gateway{
		Options:     opt,
		proxy:       newUpstreamProxy(opt.Upstream, opt.UserHeader),
		nodeModules: make(map[string]bool, len(opt.NodeModules)),
		client: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	for _, route := range opt.QueryTokenRoutes {
		g.queryRoutes = append(g.queryRoutes, strings.Split(route, "/"))
	}
	for _, module := range opt.NodeModules {
		g.nodeModules[module] = true
	}
	g.mux = g.routes()
	// 入口处先清理并重建内部身份头，之后的处理只会看到网关自己写入的身份
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.TrustIdentity(r)
		g.mux.ServeHTTP(w, r)
	})
}

// newUpstreamProxy 创建转发到中控自己 isrvd 的反向代理。
func newUpstreamProxy(upstream *url.URL, userHeader string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(upstream)
			pr.SetXForwarded()
			if user := pr.In.Header.Get(userHeader); user != "" {
				pr.Out.Header.Set(userHeader, user)
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
}

// nodeHandlers 节点管理接口，键为 service/node.PermissionRoutes 中的「方法 路径」。
// 两者是否一一对应由测试保证；即使漏配，也因权限索引里没有该路由而被拒绝（见 app.GatewayAuthorize）。
func (g *gateway) nodeHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"GET /api/node/nodes":       g.nodeList,
		"POST /api/node":            g.nodeCreate,
		"PUT /api/node/:id":         g.nodeUpdate,
		"POST /api/node/:id/revoke": g.nodeRevoke,
		"DELETE /api/node/:id":      g.nodeDelete,
		"PUT /api/node/:id/portmap": g.nodePortMapUpdate,
	}
}

func (g *gateway) routes() *http.ServeMux {
	mux := http.NewServeMux()

	for route, h := range g.nodeHandlers() {
		method, path, _ := strings.Cut(route, " ")
		mux.HandleFunc(helper.MuxPattern(method, path), g.authorized(h))
	}

	// 受管机接入：凭节点令牌连接，首次连接时待接入的令牌自动激活为节点
	mux.Handle("GET /api/node/connect", g.nodeConnect())

	// 以某个节点的视角访问：/n/<节点ID>/ 提供页面，/n/<节点ID>/api/ 转发业务接口
	mux.HandleFunc("/n/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/n/"+r.PathValue("id")+"/", http.StatusFound)
	})
	mux.HandleFunc("/n/{id}/{rest...}", g.nodePath)

	mux.Handle("/", g.proxy)
	return mux
}
