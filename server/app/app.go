package app

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	"isrvd/server/service/account"
	"isrvd/server/service/apisix"
	"isrvd/server/service/caddy"
	"isrvd/server/service/compose"
	"isrvd/server/service/copilot"
	"isrvd/server/service/cron"
	"isrvd/server/service/docker"
	"isrvd/server/service/filer"
	"isrvd/server/service/monitor"
	"isrvd/server/service/notify"
	"isrvd/server/service/overview"
	"isrvd/server/service/shell"
	"isrvd/server/service/swarm"
	"isrvd/server/service/system"
	"isrvd/server/service/webssh"

	"isrvd/public"
	"isrvd/server/config"
)

const APINamespace = "/api"

const (
	AccessAnon RouteAccess = -1 // 匿名
	AccessPerm RouteAccess = 0  // 需要具体权限
	AccessAuth RouteAccess = 1  // 登录即可访问
)

const (
	AuditIgnore   AuditLevel = -1 // 忽略
	AuditByMethod AuditLevel = 0  // 按 Method 审计
	AuditAlways   AuditLevel = 1  // 强制审计
)

// App 应用实例，持有各业务服务
type App struct {
	*gin.Engine
	servicesMu       sync.RWMutex
	requestCtxMu     sync.RWMutex
	cleanupWG        sync.WaitGroup
	lifecycleCtx     context.Context
	lifecycleCancel  context.CancelFunc
	servicesCtx      context.Context
	servicesCancel   context.CancelFunc
	requestsCtx      context.Context
	requestsCancel   context.CancelFunc
	wsConfig         *websocket.ServerConfig
	monitorCollector *monitor.Collector
	faultWatcher     *notify.FaultWatcher
	overviewSvc      *overview.Service
	configSvc        *system.ConfigService
	auditSvc         *system.AuditService
	accountSvc       *account.Service
	filerSvc         *filer.Service
	apisixSvc        *apisix.Service
	caddySvc         *caddy.Service
	dockerSvc        *docker.Service
	swarmSvc         *swarm.Service
	composeSvc       *compose.Service
	cronSvc          *cron.Service
	copilotSvc       *copilot.Service
	shellSvc         *shell.Service
	websshSvc        *webssh.Service
	probes           map[string]func(context.Context) bool // 已就绪的外部依赖模块（key 为路由 Module）→ 探活函数
	routeIndex       map[string]Route                      // METHOD+完整路径 → 路由索引
}

// RouteAccess 路由访问级别
type RouteAccess int

// AuditLevel 审计级别
type AuditLevel int

// Route 定义单个路由的完整信息（同时用于注册、权限验证和审计控制）
type Route struct {
	Key        string          `json:"key,omitempty"` // "METHOD /api/path"
	Method     string          `json:"-"`             // HTTP 方法：GET/POST/PUT/PATCH/DELETE/ANY
	Path       string          `json:"-"`             // 路由路径（Gin 格式，支持 :param 和 *）
	Handler    gin.HandlerFunc `json:"-"`             // 处理函数
	Module     string          `json:"module"`        // 模块名，空字符串表示无需模块权限
	Label      string          `json:"label"`         // 模块显示名，用于错误提示
	Access     RouteAccess     `json:"access"`        // 访问级别，0：需要具体权限，-1：匿名，1：登录即可访问
	Audit      AuditLevel      `json:"-"`             // 审计级别，0：按 Method 审计，-1：忽略，1：强制审计
	QueryToken bool            `json:"-"`             // 允许从 query ?token= 提取 JWT（用于 SSE/文件下载等无法携带 Header 的场景）
}

func StartApp() {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	servicesCtx, servicesCancel := context.WithCancel(lifecycleCtx)
	requestsCtx, requestsCancel := context.WithCancel(lifecycleCtx)
	app := &App{
		Engine:          httpd.Engine(config.Current().Server.Debug),
		lifecycleCtx:    lifecycleCtx,
		lifecycleCancel: lifecycleCancel,
		servicesCtx:     servicesCtx,
		servicesCancel:  servicesCancel,
		requestsCtx:     requestsCtx,
		requestsCancel:  requestsCancel,
		wsConfig: &websocket.ServerConfig{
			AllowedOrigins: config.Current().Server.AllowedOrigins,
		},
		routeIndex: make(map[string]Route),
	}

	app.initServices(app.servicesCtx)

	app.initRoutes()

	server := &http.Server{
		Addr:              config.Current().Server.ListenAddr,
		Handler:           app.Engine,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	app.watchReload(server)
}

// initRoutes 注册所有路由，服务可用性由 serviceAvailableMiddleware 动态检查
func (app *App) initRoutes() {
	r := app.Group(APINamespace)

	// 全局中间件。生命周期锁必须先于读取可热更新的 WebSocket/CORS 配置。
	r.Use(app.serviceLifecycleMiddleware())
	r.Use(app.wsConfig.CorsMiddleware())
	r.Use(securityHeadersMiddleware())
	// 先认证设置 username，再检查服务可用性；权限检查只在服务可用时执行。
	r.Use(app.authMiddleware(app.routeIndex))
	r.Use(app.serviceAvailableMiddleware())
	r.Use(app.permMiddleware(app.routeIndex))
	r.Use(app.auditMiddleware(app.routeIndex))

	// 注册所有声明的模块路由
	for _, route := range app.collectRoutes() {
		app.registerRoute(r, route)
	}

	// NoRoute: /api/* 返回 JSON 404，其他路径走静态文件 + SPA fallback
	staticHandler := httpd.StaticServe(http.FS(public.Efs), "")
	app.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// 非 API 路径，直接返回 404
		if strings.HasPrefix(path, APINamespace) {
			respondError(c, http.StatusNotFound, "api not found")
			return
		}
		// OpenAPI 文档默认关闭，未在配置中显式开启时不对外提供
		if !config.Current().Server.OpenAPI && strings.HasPrefix(path, "/openapi") {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		staticHandler(c)
	})
}

// collectRoutes 收集所有模块的路由定义
func (app *App) collectRoutes() []Route {
	var routes []Route
	routes = append(routes, app.defineOverviewRoutes()...)
	routes = append(routes, app.defineSystemRoutes()...)
	routes = append(routes, app.defineAccountRoutes()...)
	routes = append(routes, app.defineShellRoutes()...)
	routes = append(routes, app.defineLocalRoutes()...)
	routes = append(routes, app.defineWebSSHRoutes()...)
	routes = append(routes, app.defineFilerRoutes()...)
	routes = append(routes, app.defineCopilotRoutes()...)
	routes = append(routes, app.defineApisixRoutes()...)
	routes = append(routes, app.defineCaddyRoutes()...)
	routes = append(routes, app.defineDockerRoutes()...)
	routes = append(routes, app.defineSwarmRoutes()...)
	routes = append(routes, app.defineComposeRoutes()...)
	routes = append(routes, app.defineCronRoutes()...)
	return routes
}

// registerRoute 注册单个路由并建立索引
func (app *App) registerRoute(group *gin.RouterGroup, route Route) {
	key := route.Method + " " + APINamespace + route.Path
	route.Key = key
	app.routeIndex[key] = route

	switch route.Method {
	case "GET":
		group.GET(route.Path, route.Handler)
	case "POST":
		group.POST(route.Path, route.Handler)
	case "PUT":
		group.PUT(route.Path, route.Handler)
	case "PATCH":
		group.PATCH(route.Path, route.Handler)
	case "DELETE":
		group.DELETE(route.Path, route.Handler)
	case "ANY":
		group.Any(route.Path, route.Handler)
	}
}

func (app *App) serveWebSocket(c *gin.Context, serve func(*websocket.ServerConn)) {
	ctx := c.Request.Context()
	app.wsConfig.Handler(func(conn *websocket.ServerConn) {
		stop := context.AfterFunc(ctx, func() { _ = conn.Conn.Close() })
		defer stop()
		serve(conn)
	})(c)
}
