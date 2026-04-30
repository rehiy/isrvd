package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	svcApisix "isrvd/internal/service/apisix"
	svcCompose "isrvd/internal/service/compose"
	svcDocker "isrvd/internal/service/docker"
	svcSwarm "isrvd/internal/service/swarm"
	svcSystem "isrvd/internal/service/system"

	"isrvd/public"
)

const apiPrefix = "/api"

// App 应用实例，持有各业务服务
type App struct {
	*gin.Engine
	systemSvc   *svcSystem.Service
	settingsSvc *svcSystem.SettingsService
	memberSvc   *svcSystem.MemberService
	authSvc     *svcSystem.AuthService
	apisixSvc   *svcApisix.Service
	dockerSvc   *svcDocker.Service
	swarmSvc    *svcSwarm.Service
	composeSvc  *svcCompose.DeployService
	routePerms  map[string]Route // METHOD+完整路径 → Route 权限索引
}

func StartApp() {
	app := &App{Engine: httpd.Engine(config.Debug)}

	// 初始化各业务服务
	app.systemSvc = svcSystem.NewService()
	app.settingsSvc = svcSystem.NewSettingsService()
	app.memberSvc = svcSystem.NewMemberService()
	app.authSvc = svcSystem.NewAuthService()

	if apisixSvc, err := svcApisix.NewService(); err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
	} else {
		app.apisixSvc = apisixSvc
	}

	if dockerSvc, err := svcDocker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
	} else {
		app.dockerSvc = dockerSvc
		app.swarmSvc = svcSwarm.NewService()
	}

	if composeSvc, err := svcCompose.NewService(); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
	} else {
		app.composeSvc = composeSvc
	}

	// 统一注册路由
	app.initRoutes()
	httpd.StaticEmbed(public.Efs, "", "")

	// 启动 HTTP 服务
	httpd.Server(config.ListenAddr)
}

// Route 定义单个路由的完整信息（同时用于注册和权限验证）
type Route struct {
	Method  string          // HTTP 方法：GET/POST/PUT/PATCH/DELETE/ANY
	Path    string          // 路由路径（Gin 格式，支持 :param 和 *）
	Handler gin.HandlerFunc // 处理函数
	Module  string          // 模块名，空字符串表示无需模块权限（如 /auth/*）
	Label   string          // 模块显示名，用于错误提示
	Perm    string          // 所需权限：空或"r"=只读，"rw"=读写
}

// initRoutes 初始化路由表并注册所有路由
// 同时完成路由注册和权限配置，实现真正的集中式管理
func (app *App) initRoutes() {
	r := app.Group(apiPrefix)

	// Auth 相关（无需权限验证，单独注册）
	r.GET("/auth/info", MixAuthMiddleware(), app.authInfo)
	r.POST("/auth/login", app.login)

	// 加载路由表
	allRoutes := app.defineRoutes()
	app.routePerms = make(map[string]Route, len(allRoutes))

	// 按条件注册路由（处理服务可用性依赖）
	protected := r.Group("")
	protected.Use(AuthMiddleware())
	protected.Use(app.RoutePermMiddleware())
	for _, route := range allRoutes {
		if !app.isRouteAvailable(route) {
			continue
		}
		app.registerRoute(protected, route)
	}
}

// registerRoute 注册单个路由，并同步建立 METHOD+完整路由模板的权限索引
func (app *App) registerRoute(group *gin.RouterGroup, route Route) {
	app.routePerms[route.Method+" "+apiPrefix+route.Path] = route

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
	case "apisix":
		return app.apisixSvc != nil
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

// defineRoutes 定义所有需要认证+权限验证的路由表
// 将路由定义从 initRoutes 中分离，使职责更清晰
func (app *App) defineRoutes() []Route {
	return []Route{
		// System - 只读
		{Method: "GET", Path: "/system/stats", Handler: app.systemStat, Module: "system", Label: "系统管理", Perm: "r"},
		{Method: "GET", Path: "/system/probe", Handler: app.systemProbe, Module: "system", Label: "系统管理", Perm: "r"},
		{Method: "GET", Path: "/system/settings", Handler: app.systemGetSettings, Module: "system", Label: "系统管理", Perm: "r"},
		{Method: "GET", Path: "/system/members", Handler: app.systemListMembers, Module: "system", Label: "系统管理", Perm: "r"},

		// System - 读写
		{Method: "PUT", Path: "/system/settings", Handler: app.systemUpdateSettings, Module: "system", Label: "系统管理", Perm: "rw"},
		{Method: "POST", Path: "/system/members", Handler: app.systemCreateMember, Module: "system", Label: "系统管理", Perm: "rw"},
		{Method: "PUT", Path: "/system/member/:username", Handler: app.systemUpdateMember, Module: "system", Label: "系统管理", Perm: "rw"},
		{Method: "DELETE", Path: "/system/member/:username", Handler: app.systemDeleteMember, Module: "system", Label: "系统管理", Perm: "rw"},

		// Filer - 只读
		{Method: "POST", Path: "/filer/list", Handler: app.filerList, Module: "filer", Label: "文件管理", Perm: "r"},
		{Method: "POST", Path: "/filer/download", Handler: app.filerDownload, Module: "filer", Label: "文件管理", Perm: "r"},
		{Method: "POST", Path: "/filer/read", Handler: app.filerRead, Module: "filer", Label: "文件管理", Perm: "r"},

		// Filer - 读写
		{Method: "POST", Path: "/filer/upload", Handler: app.filerUpload, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/delete", Handler: app.filerDelete, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/mkdir", Handler: app.filerMkdir, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/create", Handler: app.filerCreate, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/modify", Handler: app.filerModify, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/rename", Handler: app.filerRename, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/chmod", Handler: app.filerChmod, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/zip", Handler: app.filerZip, Module: "filer", Label: "文件管理", Perm: "rw"},
		{Method: "POST", Path: "/filer/unzip", Handler: app.filerUnzip, Module: "filer", Label: "文件管理", Perm: "rw"},

		// Agent
		{Method: "ANY", Path: "/agent/proxy/*path", Handler: app.agentProxy, Module: "agent", Label: "AI Agent", Perm: "r"},

		// Apisix - 只读
		{Method: "GET", Path: "/apisix/routes", Handler: app.apisixListRoutes, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/route/:id", Handler: app.apisixGetRoute, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/consumers", Handler: app.apisixListConsumers, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/plugin_configs", Handler: app.apisixListPluginConfigs, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/plugin_config/:id", Handler: app.apisixGetPluginConfig, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/plugins", Handler: app.apisixListPlugins, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/upstreams", Handler: app.apisixListUpstreams, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/upstream/:id", Handler: app.apisixGetUpstream, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/ssls", Handler: app.apisixListSSLs, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/ssl/:id", Handler: app.apisixGetSSL, Module: "apisix", Label: "APISIX", Perm: "r"},
		{Method: "GET", Path: "/apisix/whitelist", Handler: app.apisixGetWhitelist, Module: "apisix", Label: "APISIX", Perm: "r"},

		// Apisix - 读写
		{Method: "POST", Path: "/apisix/routes", Handler: app.apisixCreateRoute, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PUT", Path: "/apisix/route/:id", Handler: app.apisixUpdateRoute, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PATCH", Path: "/apisix/route/:id/status", Handler: app.apisixPatchRouteStatus, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "DELETE", Path: "/apisix/route/:id", Handler: app.apisixDeleteRoute, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "POST", Path: "/apisix/consumers", Handler: app.apisixCreateConsumer, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PUT", Path: "/apisix/consumer/:username", Handler: app.apisixUpdateConsumer, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "DELETE", Path: "/apisix/consumer/:username", Handler: app.apisixDeleteConsumer, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "POST", Path: "/apisix/plugin_configs", Handler: app.apisixCreatePluginConfig, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PUT", Path: "/apisix/plugin_config/:id", Handler: app.apisixUpdatePluginConfig, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "DELETE", Path: "/apisix/plugin_config/:id", Handler: app.apisixDeletePluginConfig, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "POST", Path: "/apisix/upstreams", Handler: app.apisixCreateUpstream, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PUT", Path: "/apisix/upstream/:id", Handler: app.apisixUpdateUpstream, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "DELETE", Path: "/apisix/upstream/:id", Handler: app.apisixDeleteUpstream, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "POST", Path: "/apisix/ssls", Handler: app.apisixCreateSSL, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "PUT", Path: "/apisix/ssl/:id", Handler: app.apisixUpdateSSL, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "DELETE", Path: "/apisix/ssl/:id", Handler: app.apisixDeleteSSL, Module: "apisix", Label: "APISIX", Perm: "rw"},
		{Method: "POST", Path: "/apisix/whitelist/revoke", Handler: app.apisixRevokeWhitelist, Module: "apisix", Label: "APISIX", Perm: "rw"},

		// Docker - 只读
		{Method: "GET", Path: "/docker/info", Handler: app.dockerInfo, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/containers", Handler: app.dockerListContainers, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/container/:id/logs", Handler: app.dockerContainerLogs, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/container/:id/stats", Handler: app.dockerContainerStats, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/images", Handler: app.dockerListImages, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/image/:id", Handler: app.dockerInspectImage, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/image/search/:term", Handler: app.dockerSearchImages, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/networks", Handler: app.dockerListNetworks, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/network/:id", Handler: app.dockerNetworkInspect, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/volumes", Handler: app.dockerListVolumes, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/volume/:name", Handler: app.dockerVolumeInspect, Module: "docker", Label: "Docker", Perm: "r"},
		{Method: "GET", Path: "/docker/registries", Handler: app.dockerListRegistries, Module: "docker", Label: "Docker", Perm: "r"},

		// Docker - 读写
		{Method: "POST", Path: "/docker/container/:id/action", Handler: app.dockerContainerAction, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/container/create", Handler: app.dockerCreateContainer, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/image/:id/action", Handler: app.dockerImageAction, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/image/tag", Handler: app.dockerTagImage, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/image/build", Handler: app.dockerBuildImage, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/network/:id/action", Handler: app.dockerNetworkAction, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/network/create", Handler: app.dockerCreateNetwork, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/volume/:name/action", Handler: app.dockerVolumeAction, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/volume/create", Handler: app.dockerCreateVolume, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/registries", Handler: app.dockerCreateRegistry, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "PUT", Path: "/docker/registries", Handler: app.dockerUpdateRegistry, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "DELETE", Path: "/docker/registries", Handler: app.dockerDeleteRegistry, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/registry/push", Handler: app.dockerPushImage, Module: "docker", Label: "Docker", Perm: "rw"},
		{Method: "POST", Path: "/docker/registry/pull", Handler: app.dockerPullFromRegistry, Module: "docker", Label: "Docker", Perm: "rw"},

		// Swarm - 只读
		{Method: "GET", Path: "/swarm/info", Handler: app.swarmInfo, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/nodes", Handler: app.swarmListNodes, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/node/:id", Handler: app.swarmInspectNode, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/join-tokens", Handler: app.swarmGetJoinTokens, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/services", Handler: app.swarmListServices, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/service/:id", Handler: app.swarmInspectService, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/service/:id/logs", Handler: app.swarmServiceLogs, Module: "swarm", Label: "Swarm", Perm: "r"},
		{Method: "GET", Path: "/swarm/tasks", Handler: app.swarmListTasks, Module: "swarm", Label: "Swarm", Perm: "r"},

		// Swarm - 读写
		{Method: "POST", Path: "/swarm/node/:id/action", Handler: app.NodeDTOAction, Module: "swarm", Label: "Swarm", Perm: "rw"},
		{Method: "POST", Path: "/swarm/service/create", Handler: app.swarmCreateService, Module: "swarm", Label: "Swarm", Perm: "rw"},
		{Method: "POST", Path: "/swarm/service/:id/action", Handler: app.swarmServiceAction, Module: "swarm", Label: "Swarm", Perm: "rw"},
		{Method: "POST", Path: "/swarm/service/:id/redeploy", Handler: app.swarmForceUpdateService, Module: "swarm", Label: "Swarm", Perm: "rw"},

		// Compose - 只读
		{Method: "GET", Path: "/compose/docker/:name", Handler: app.composeGetDockerContent, Module: "compose", Label: "Compose", Perm: "r"},
		{Method: "GET", Path: "/compose/swarm/:name", Handler: app.composeGetSwarmContent, Module: "compose", Label: "Compose", Perm: "r"},

		// Compose - 读写
		{Method: "POST", Path: "/compose/docker/deploy", Handler: app.composeDeployDocker, Module: "compose", Label: "Compose", Perm: "rw"},
		{Method: "PUT", Path: "/compose/docker/:name", Handler: app.composeRedeployDocker, Module: "compose", Label: "Compose", Perm: "rw"},
		{Method: "POST", Path: "/compose/swarm/deploy", Handler: app.composeDeploySwarm, Module: "compose", Label: "Compose", Perm: "rw"},
		{Method: "PUT", Path: "/compose/swarm/:name", Handler: app.composeRedeploySwarm, Module: "compose", Label: "Compose", Perm: "rw"},

		// WebSocket - 读写
		{Method: "GET", Path: "/ws/shell", Handler: app.shellWebSocket, Module: "shell", Label: "Shell终端", Perm: "rw"},
		{Method: "GET", Path: "/ws/docker/exec", Handler: app.dockerContainerExec, Module: "docker", Label: "Docker", Perm: "rw"},
	}
}
