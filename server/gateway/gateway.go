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

	"isrvd/server/config"
	"isrvd/server/service/account"
	"isrvd/server/service/node"
)

// Options 中控网关的装配参数
type Options struct {
	Upstream         *url.URL        // 同进程 isrvd 的回环地址
	Nodes            *node.Service   // 节点服务
	AllowedOrigins   func() []string // isrvd 配置的跨域白名单，用于校验转发到节点的 WebSocket；每次校验时读取，配置重载后即时生效
	QueryTokenRoutes []string        // 允许 ?token= 认证的 GET 路由，来自 app.QueryTokenRoutes
	NodeModules      []string        // 节点视角下转发给受管机的业务模块，来自 app.NodeModules
	Audit            Auditor         // 审计入口；节点管理与节点上的写操作、终端会话按 isrvd 的审计基线记录

	// Authorize 复用应用当前账户服务认证调用者并校验路由权限，返回用户名；
	// 失败时返回应写出的 HTTP 状态码及错误。
	Authorize func(r *http.Request, allowQueryToken bool) (string, int, error)
}

type gateway struct {
	nodes    *node.Service
	upstream *url.URL
	proxy    *httputil.ReverseProxy // 转发到中控自己的 isrvd
	client   *http.Client           // 仅用于页面 bootstrap 聚合

	allowedOrigins func() []string // 额外允许的 WebSocket Origin（isrvd 的 allowedOrigins），默认仅同源
	audit          Auditor
	authorize      func(*http.Request, bool) (string, int, error)

	mux *http.ServeMux // 对外路由表；节点视角下的 /api/node/* 也经它分发

	queryRoutes [][]string      // 允许 ?token= 的 GET 路由，按路径段拆分
	nodeModules map[string]bool // 节点视角下转发给受管机的业务模块
}

// Handler 返回中控网关：节点管理接口与 /n/<节点ID>/ 下的节点转发由它处理，
// 其余请求（含全部页面）原样转发给 upstream（同进程的 isrvd，只监听回环地址）。
// 节点切换与节点管理页面由 webview 实现，通过 isrvd 提供。
func Handler(opt Options) http.Handler {
	g := &gateway{
		nodes:          opt.Nodes,
		upstream:       opt.Upstream,
		allowedOrigins: opt.AllowedOrigins,
		audit:          opt.Audit,
		authorize:      opt.Authorize,
		nodeModules:    make(map[string]bool, len(opt.NodeModules)),
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
	g.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(opt.Upstream)
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
	return g.routes()
}

func (g *gateway) routes() http.Handler {
	mux := http.NewServeMux()

	// 节点管理：路由与可分配权限共用 service/node 的元信息。
	handlers := map[string]http.HandlerFunc{
		"GET /api/node/nodes":       g.nodeList,
		"POST /api/node":            g.nodeCreate,
		"PUT /api/node/:id":         g.nodeUpdate,
		"POST /api/node/:id/revoke": g.nodeRevoke,
		"DELETE /api/node/:id":      g.nodeDelete,
		"PUT /api/node/:id/portmap": g.nodePortMapUpdate,
	}
	// 两份清单必须一一对应：缺处理函数会在请求到达时 panic，多出的处理函数则静默不生效，都在启动期暴露。
	routes := node.PermissionRoutes()
	if len(routes) != len(handlers) {
		panic("节点管理路由与处理函数数量不一致")
	}
	for _, route := range routes {
		h := handlers[route.Method+" "+route.Path]
		if h == nil {
			panic("节点管理路由缺少处理函数: " + route.Method + " " + route.Path)
		}
		mux.HandleFunc(account.MuxPattern(route.Method, route.Path), g.permissionOnly(h))
	}

	// 受管机接入：凭节点令牌连接，首次连接时待接入的令牌自动激活为节点
	mux.Handle("GET /api/node/connect", g.nodeConnect())

	// 以某个节点的视角访问：/n/<节点ID>/ 提供页面，/n/<节点ID>/api/ 转发业务接口
	mux.HandleFunc("/n/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/n/"+r.PathValue("id")+"/", http.StatusFound)
	})
	mux.HandleFunc("/n/{id}/{rest...}", g.nodePath)

	mux.Handle("/", g.proxy)
	g.mux = mux
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
		mux.ServeHTTP(w, r)
	})
}
