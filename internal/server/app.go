package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"

	svcAccount "isrvd/internal/service/account"
	svcApisix "isrvd/internal/service/apisix"
	svcCaddy "isrvd/internal/service/caddy"
	svcCompose "isrvd/internal/service/compose"
	svcCron "isrvd/internal/service/cron"
	svcDocker "isrvd/internal/service/docker"
	svcFiler "isrvd/internal/service/filer"
	svcOverview "isrvd/internal/service/overview"
	svcSwarm "isrvd/internal/service/swarm"
	svcSystem "isrvd/internal/service/system"

	"isrvd/config"
	"isrvd/internal/registry"
	"isrvd/public"
)

// App 应用实例，持有各业务服务
type App struct {
	*gin.Engine
	wsConfig    *websocket.ServerConfig
	overviewSvc *svcOverview.Service
	configSvc   *svcSystem.ConfigService
	auditSvc    *svcSystem.AuditService
	accountSvc  *svcAccount.Service
	filerSvc    *svcFiler.Service
	apisixSvc   *svcApisix.Service
	caddySvc    *svcCaddy.Service
	dockerSvc   *svcDocker.Service
	swarmSvc    *svcSwarm.Service
	composeSvc  *svcCompose.Service
	cronSvc     *svcCron.Service
	routeIndex  map[string]Route // METHOD+完整路径 → 路由索引
}

// RouteAccess 路由访问级别
type RouteAccess int

// AuditLevel 审计级别
type AuditLevel int

// Route 定义单个路由的完整信息（同时用于注册、权限验证和审计控制）
type Route struct {
	Key     string          `json:"key,omitempty"` // "METHOD /api/path"
	Method  string          `json:"-"`             // HTTP 方法：GET/POST/PUT/PATCH/DELETE/ANY
	Path    string          `json:"-"`             // 路由路径（Gin 格式，支持 :param 和 *）
	Handler gin.HandlerFunc `json:"-"`             // 处理函数
	Module  string          `json:"module"`        // 模块名，空字符串表示无需模块权限
	Label   string          `json:"label"`         // 模块显示名，用于错误提示
	Access  RouteAccess     `json:"access"`        // 访问级别，0：需要具体权限，-1：匿名，1：登录即可访问
	Audit   AuditLevel      `json:"-"`             // 审计级别，0：按 Method 审计，-1：忽略，1：强制审计
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

	// 初始化各业务服务
	app.overviewSvc = svcOverview.NewService()
	app.configSvc = svcSystem.NewConfigService()
	app.auditSvc = svcSystem.NewAuditService()
	app.accountSvc = svcAccount.NewService()
	app.filerSvc = svcFiler.NewService()

	if apisixSvc, err := svcApisix.NewService(); err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
	} else {
		app.apisixSvc = apisixSvc
	}

	if caddySvc, err := svcCaddy.NewService(); err != nil {
		logman.Warn("Caddy service unavailable", "error", err)
	} else {
		app.caddySvc = caddySvc
	}

	if dockerSvc, err := svcDocker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
	} else {
		app.dockerSvc = dockerSvc
		if swarmSvc, err := svcSwarm.NewService(); err != nil {
			logman.Warn("Swarm service unavailable", "error", err)
		} else {
			app.swarmSvc = swarmSvc
		}
	}

	if composeSvc, err := svcCompose.NewService(); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
	} else {
		app.composeSvc = composeSvc
	}

	app.cronSvc = svcCron.NewService(registry.DockerService)

	// 统一注册路由
	app.initRoutes()
	httpd.StaticEmbed(public.Efs, "", "")

	// 启动 HTTP 服务
	httpd.Server(config.Server.ListenAddr)
}

// initRoutes 初始化路由表并注册所有路由
// 按模块注册路由，每个模块自己管理路由定义和注册
func (app *App) initRoutes() {
	r := app.Group(APINamespace)

	// CORS 中间件（必须在最前面）
	r.Use(app.wsConfig.CorsMiddleware())

	// 安全响应头中间件
	r.Use(securityHeadersMiddleware())

	// 认证、权限与审计中间件
	r.Use(AuthMiddleware(app.routeIndex, app.accountSvc))
	r.Use(PermMiddleware(app.routeIndex, app.accountSvc))
	r.Use(AuditMiddleware(app.routeIndex, app.auditSvc))

	// 加载所有模块的路由定义并注册
	for _, route := range app.collectRoutes() {
		if app.isRouteAvailable(route) {
			app.registerRoute(r, route)
		}
	}
}

// collectRoutes 收集所有模块的路由定义
// 每个模块通过 defineXxxRoutes() 方法返回自己的路由列表
func (app *App) collectRoutes() []Route {
	var routes []Route

	// 概览（系统统计 + 服务探测，无需权限）
	routes = append(routes, app.defineOverviewRoutes()...)
	// 系统设置
	routes = append(routes, app.defineSystemRoutes()...)
	// Account 模块
	routes = append(routes, app.defineAccountRoutes()...)
	// Web 终端
	routes = append(routes, app.defineShellRoutes()...)
	// 文件管理
	routes = append(routes, app.defineFilerRoutes()...)
	// LLM 代理
	routes = append(routes, app.defineAgentRoutes()...)
	// APISIX 管理
	routes = append(routes, app.defineApisixRoutes()...)
	// Caddy 管理
	routes = append(routes, app.defineCaddyRoutes()...)
	// Docker 管理
	routes = append(routes, app.defineDockerRoutes()...)
	// Swarm 管理
	routes = append(routes, app.defineSwarmRoutes()...)
	// Compose 部署
	routes = append(routes, app.defineComposeRoutes()...)
	// 计划任务
	routes = append(routes, app.defineCronRoutes()...)

	return routes
}

// registerRoute 注册单个路由，并同步建立 METHOD+完整路由模板索引
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

// isRouteAvailable 检查路由是否满足服务可用性条件（基于 Module 字段判断）
func (app *App) isRouteAvailable(route Route) bool {
	switch route.Module {
	case "agent":
		return config.Agent.BaseURL != ""
	case "apisix":
		return app.apisixSvc != nil
	case "caddy":
		return app.caddySvc != nil
	case "docker", "shell":
		return app.dockerSvc != nil
	case "swarm":
		return app.dockerSvc != nil && app.swarmSvc != nil
	case "compose":
		return app.dockerSvc != nil && app.composeSvc != nil
	default:
		return true
	}
}
