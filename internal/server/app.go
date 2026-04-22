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

// App 应用实例，持有各业务服务
type App struct {
	*gin.Engine
	apisixSvc   *svcApisix.Service
	dockerSvc   *svcDocker.Service
	swarmSvc    *svcSwarm.Service
	composeSvc  *svcCompose.DeployService
	systemSvc   *svcSystem.Service
	settingsSvc *svcSystem.SettingsService
	memberSvc   *svcSystem.MemberService
	authSvc     *svcSystem.AuthService
}

func NewApp() *App {
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

	if composeSvc, err := svcCompose.NewComposeService(); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
	} else {
		app.composeSvc = composeSvc
	}

	// 注册路由
	app.setupRouter()

	logman.Info("Server starting",
		"listenAddr", config.ListenAddr,
		"rootDirectory", config.RootDirectory,
		"members", len(config.Members),
	)

	httpd.StaticEmbed(public.Efs, "", "")
	httpd.Server(config.ListenAddr)

	return app
}

// setupRouter 注册所有路由
func (app *App) setupRouter() {
	api := app.Group("/api")

	api.GET("/auth/info", MixAuthMiddleware(), app.authInfo)
	api.POST("/auth/login", app.login)

	authApi := api.Group("")
	authApi.Use(AuthMiddleware())

	authApi.POST("/auth/logout", app.logout)

	// 文件管理（只读）
	fr := authApi.Group("/filer")
	fr.Use(PermMiddleware("filer", false))
	{
		fr.POST("/list", app.filerList)
		fr.POST("/download", app.filerDownload)
		fr.POST("/read", app.filerRead)
	}
	// 文件管理（读写）
	fw := authApi.Group("/filer")
	fw.Use(PermMiddleware("filer", true))
	{
		fw.POST("/upload", app.filerUpload)
		fw.POST("/delete", app.filerDelete)
		fw.POST("/mkdir", app.filerMkdir)
		fw.POST("/create", app.filerCreate)
		fw.POST("/modify", app.filerModify)
		fw.POST("/rename", app.filerRename)
		fw.POST("/chmod", app.filerChmod)
		fw.POST("/zip", app.filerZip)
		fw.POST("/unzip", app.filerUnzip)
	}

	// Agent LLM 代理
	agentGroup := authApi.Group("/agent")
	agentGroup.Use(PermMiddleware("agent", false))
	{
		agentGroup.Any("/proxy/*path", app.agentProxy)
	}

	// Apisix
	if app.apisixSvc != nil {
		// Apisix 只读
		ar := authApi.Group("/apisix")
		ar.Use(PermMiddleware("apisix", false))
		{
			ar.GET("/routes", app.apisixListRoutes)
			ar.GET("/route/:id", app.apisixGetRoute)
			ar.GET("/consumers", app.apisixListConsumers)
			ar.GET("/plugin_configs", app.apisixListPluginConfigs)
			ar.GET("/plugins", app.apisixListPlugins)
			ar.GET("/upstreams", app.apisixListUpstreams)
			ar.GET("/whitelist", app.apisixGetWhitelist)
		}
		// Apisix 读写
		aw := authApi.Group("/apisix")
		aw.Use(PermMiddleware("apisix", true))
		{
			aw.POST("/routes", app.apisixCreateRoute)
			aw.PUT("/route/:id", app.apisixUpdateRoute)
			aw.PATCH("/route/:id/status", app.apisixPatchRouteStatus)
			aw.DELETE("/route/:id", app.apisixDeleteRoute)
			aw.POST("/consumers", app.apisixCreateConsumer)
			aw.PUT("/consumer/:username", app.apisixUpdateConsumer)
			aw.DELETE("/consumer/:username", app.apisixDeleteConsumer)
			aw.PUT("/whitelist/revoke", app.apisixRevokeWhitelist)
		}
	}

	// Docker
	if app.dockerSvc != nil {
		// Docker 只读
		dr := authApi.Group("/docker")
		dr.Use(PermMiddleware("docker", false))
		{
			dr.GET("/info", app.dockerInfo)
			dr.GET("/containers", app.dockerListContainers)
			dr.GET("/container/:id/logs", app.dockerContainerLogs)
			dr.GET("/container/:id/stats", app.dockerContainerStats)
			dr.GET("/images", app.dockerListImages)
			dr.GET("/image/:id", app.dockerInspectImage)
			dr.GET("/image/search/:term", app.dockerSearchImages)
			dr.GET("/networks", app.dockerListNetworks)
			dr.GET("/network/:id", app.dockerNetworkInspect)
			dr.GET("/volumes", app.dockerListVolumes)
			dr.GET("/volume/:name", app.dockerVolumeInspect)
			dr.GET("/registries", app.dockerListRegistries)
		}
		// Docker 读写
		dw := authApi.Group("/docker")
		dw.Use(PermMiddleware("docker", true))
		{
			dw.POST("/container/:id/action", app.dockerContainerAction)
			dw.POST("/container/create", app.dockerCreateContainer)
			dw.POST("/image/:id/action", app.dockerImageAction)
			dw.POST("/image/pull", app.dockerPullImage)
			dw.POST("/image/tag", app.dockerTagImage)
			dw.POST("/image/build", app.dockerBuildImage)
			dw.POST("/network/:id/action", app.dockerNetworkAction)
			dw.POST("/network/create", app.dockerCreateNetwork)
			dw.POST("/volume/:name/action", app.dockerVolumeAction)
			dw.POST("/volume/create", app.dockerCreateVolume)
			dw.POST("/registries", app.dockerCreateRegistry)
			dw.PUT("/registries", app.dockerUpdateRegistry)
			dw.DELETE("/registries", app.dockerDeleteRegistry)
			dw.POST("/registry/push", app.dockerPushImage)
			dw.POST("/registry/pull", app.dockerPullFromRegistry)
		}
	}

	// Swarm（依赖 Docker）
	if app.dockerSvc != nil && app.swarmSvc != nil {
		// Swarm 只读
		swr := authApi.Group("/swarm")
		swr.Use(PermMiddleware("swarm", false))
		{
			swr.GET("/info", app.swarmInfo)
			swr.GET("/nodes", app.swarmListNodes)
			swr.GET("/node/:id", app.swarmInspectNode)
			swr.GET("/join-tokens", app.swarmGetJoinTokens)
			swr.GET("/services", app.swarmListServices)
			swr.GET("/service/:id", app.swarmInspectService)
			swr.GET("/service/:id/logs", app.swarmServiceLogs)
			swr.GET("/tasks", app.swarmListTasks)
		}
		// Swarm 读写
		sww := authApi.Group("/swarm")
		sww.Use(PermMiddleware("swarm", true))
		{
			sww.POST("/node/:id/action", app.NodeDTOAction)
			sww.POST("/service/create", app.swarmCreateService)
			sww.POST("/service/:id/action", app.swarmServiceAction)
			sww.POST("/service/:id/redeploy", app.swarmForceUpdateService)
		}
	}

	// Compose（依赖 Docker）
	if app.dockerSvc != nil && app.composeSvc != nil {
		// Compose 只读
		cr := authApi.Group("/compose")
		cr.Use(PermMiddleware("compose", false))
		{
			cr.GET("/docker/:name", app.composeGetDockerContent)
			cr.GET("/swarm/:name", app.composeGetSwarmContent)
		}
		// Compose 读写
		cw := authApi.Group("/compose")
		cw.Use(PermMiddleware("compose", true))
		{
			cw.POST("/docker/deploy", app.composeDeployDocker)
			cw.PUT("/docker/:name", app.composeRedeployDocker)
			cw.POST("/swarm/deploy", app.composeDeploySwarm)
			cw.PUT("/swarm/:name", app.composeRedeploySwarm)
		}
	}

	// 系统（stats/probe/health 所有登录用户可访问）
	authApi.GET("/system/stats", app.systemStat)
	authApi.GET("/system/probe", app.systemProbe)
	authApi.GET("/system/health", app.systemHealth)
	// 系统设置与成员管理需要 system 权限
	sysr := authApi.Group("/system")
	sysr.Use(PermMiddleware("system", false))
	{
		sysr.GET("/settings", app.systemGetSettings)
		sysr.GET("/members", app.systemListMembers)
	}
	sysw := authApi.Group("/system")
	sysw.Use(PermMiddleware("system", true))
	{
		sysw.PUT("/settings", app.systemUpdateSettings)
		sysw.POST("/members", app.systemCreateMember)
		sysw.PUT("/member/:username", app.systemUpdateMember)
		sysw.DELETE("/member/:username", app.systemDeleteMember)
	}

	// WebSocket
	ws := app.Group("/ws")
	ws.Use(AuthMiddleware())
	{
		// shell 权限由 handler 内部的 allowTerminal 控制，无需额外 perm 中间件
		ws.GET("/shell", app.shellWebSocket)
		if app.dockerSvc != nil {
			ws.GET("/docker/container/exec", PermMiddleware("docker", true), app.dockerContainerExec)
		}
	}
}
