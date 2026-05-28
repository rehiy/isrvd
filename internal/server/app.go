package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	svcAccount "isrvd/internal/service/account"
	svcAgent "isrvd/internal/service/agent"
	svcApisix "isrvd/internal/service/apisix"
	svcCaddy "isrvd/internal/service/caddy"
	svcCompose "isrvd/internal/service/compose"
	svcCron "isrvd/internal/service/cron"
	svcDocker "isrvd/internal/service/docker"
	svcFiler "isrvd/internal/service/filer"
	svcMonitor "isrvd/internal/service/monitor"
	svcOverview "isrvd/internal/service/overview"
	svcShell "isrvd/internal/service/shell"
	svcSwarm "isrvd/internal/service/swarm"
	svcSystem "isrvd/internal/service/system"
	svcWebSSH "isrvd/internal/service/webssh"

	"isrvd/config"
	"isrvd/public"
)

// App 应用实例，持有各业务服务
type App struct {
	*gin.Engine
	wsConfig         *websocket.ServerConfig
	monitorCollector *svcMonitor.Collector
	overviewSvc      *svcOverview.Service
	configSvc        *svcSystem.ConfigService
	auditSvc         *svcSystem.AuditService
	accountSvc       *svcAccount.Service
	filerSvc         *svcFiler.Service
	apisixSvc        *svcApisix.Service
	caddySvc         *svcCaddy.Service
	dockerSvc        *svcDocker.Service
	swarmSvc         *svcSwarm.Service
	composeSvc       *svcCompose.Service
	cronSvc          *svcCron.Service
	agentSvc         *svcAgent.Service
	shellSvc         *svcShell.Service
	websshSvc        *svcWebSSH.Service
	routeIndex       map[string]Route // METHOD+完整路径 → 路由索引
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

func StartApp() {
	app := &App{
		Engine: httpd.Engine(config.Server.Debug),
		wsConfig: &websocket.ServerConfig{
			AllowedOrigins: config.Server.AllowedOrigins,
		},
		routeIndex: make(map[string]Route),
	}

	app.initServices()
	app.startMonitor()

	app.initRoutes()
	httpd.StaticEmbed(public.Efs, "", "")

	app.watchReload()
	httpd.Server(config.Server.ListenAddr)
}

// initRoutes 注册所有路由，服务可用性由 serviceAvailableMiddleware 动态检查
func (app *App) initRoutes() {
	r := app.Group(APINamespace)
	r.Use(app.wsConfig.CorsMiddleware())
	r.Use(securityHeadersMiddleware())
	r.Use(app.serviceAvailableMiddleware())
	r.Use(AuthMiddleware(app.routeIndex, app.accountSvc))
	r.Use(PermMiddleware(app.routeIndex, app.accountSvc))
	r.Use(AuditMiddleware(app.routeIndex, app.auditSvc))

	for _, route := range app.collectRoutes() {
		app.registerRoute(r, route)
	}
}

// collectRoutes 收集所有模块的路由定义
func (app *App) collectRoutes() []Route {
	var routes []Route
	routes = append(routes, app.defineOverviewRoutes()...)
	routes = append(routes, app.defineSystemRoutes()...)
	routes = append(routes, app.defineAccountRoutes()...)
	routes = append(routes, app.defineShellRoutes()...)
	routes = append(routes, app.defineWebSSHRoutes()...)
	routes = append(routes, app.defineFilerRoutes()...)
	routes = append(routes, app.defineAgentRoutes()...)
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
