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
	dockerSvc   *svcDocker.Service
	swarmSvc    *svcSwarm.Service
	apisixSvc   *svcApisix.Service
	composeSvc  *svcCompose.DeployService
	systemSvc   *svcSystem.Service
	settingsSvc *svcSystem.SettingsService
	memberSvc   *svcSystem.MemberService
}

func NewApp() *App {
	app := &App{Engine: httpd.Engine(config.Debug)}

	// 初始化各业务服务
	app.systemSvc = svcSystem.NewService()
	app.settingsSvc = svcSystem.NewSettingsService()
	app.memberSvc = svcSystem.NewMemberService()
	app.swarmSvc = svcSwarm.NewService()

	// compose snapshot service 先初始化，供 docker service 注入
	snapSvc := svcCompose.GetSnapshotService()

	if dockerSvc, err := svcDocker.NewService(snapSvc); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
	} else {
		app.dockerSvc = dockerSvc
	}

	if apisixSvc, err := svcApisix.NewService(); err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
	} else {
		app.apisixSvc = apisixSvc
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
	{
		api.POST("/login", app.login)

		auth := api.Group("")
		auth.Use(AuthMiddleware())
		{
			auth.POST("/logout", app.logout)

			// 文件管理
			f := auth.Group("/filer")
			{
				f.POST("/list", app.filerList)
				f.POST("/upload", app.filerUpload)
				f.POST("/download", app.filerDownload)
				f.POST("/delete", app.filerDelete)
				f.POST("/mkdir", app.filerMkdir)
				f.POST("/create", app.filerCreate)
				f.POST("/read", app.filerRead)
				f.POST("/modify", app.filerModify)
				f.POST("/rename", app.filerRename)
				f.POST("/chmod", app.filerChmod)
				f.POST("/zip", app.filerZip)
				f.POST("/unzip", app.filerUnzip)
			}

			// Agent LLM 代理
			auth.Any("/agent/proxy/*path", app.agentProxy)

			// Apisix
			if app.apisixSvc != nil {
				a := auth.Group("/apisix")
				{
					a.GET("/routes", app.apisixListRoutes)
					a.GET("/route/:id", app.apisixGetRoute)
					a.POST("/routes", app.apisixCreateRoute)
					a.PUT("/route/:id", app.apisixUpdateRoute)
					a.PATCH("/route/:id/status", app.apisixPatchRouteStatus)
					a.DELETE("/route/:id", app.apisixDeleteRoute)
					a.GET("/consumers", app.apisixListConsumers)
					a.POST("/consumers", app.apisixCreateConsumer)
					a.PUT("/consumer/:username", app.apisixUpdateConsumer)
					a.DELETE("/consumer/:username", app.apisixDeleteConsumer)
					a.GET("/plugin_configs", app.apisixListPluginConfigs)
					a.GET("/plugins", app.apisixListPlugins)
					a.GET("/upstreams", app.apisixListUpstreams)
					a.GET("/whitelist", app.apisixGetWhitelist)
					a.PUT("/whitelist/revoke", app.apisixRevokeWhitelist)
				}
			}

			// Docker
			if app.dockerSvc != nil {
				d := auth.Group("/docker")
				{
					d.GET("/info", app.dockerInfo)
					d.GET("/containers", app.dockerListContainers)
					d.POST("/container/action", app.dockerContainerAction)
					d.POST("/container/create", app.dockerCreateContainer)
					d.POST("/container/logs", app.dockerContainerLogs)
					d.GET("/container/:id/stats", app.dockerContainerStats)
					d.GET("/container/:id/config", app.dockerGetContainerConfig)
					d.POST("/container/update", app.dockerUpdateContainerConfig)
					d.GET("/images", app.dockerListImages)
					d.POST("/image/action", app.dockerImageAction)
					d.GET("/image/:id", app.dockerInspectImage)
					d.POST("/image/pull", app.dockerPullImage)
					d.POST("/image/tag", app.dockerTagImage)
					d.GET("/image/search/:term", app.dockerSearchImages)
					d.POST("/image/build", app.dockerBuildImage)
					d.GET("/networks", app.dockerListNetworks)
					d.GET("/network/:id", app.dockerNetworkInspect)
					d.POST("/network/action", app.dockerNetworkAction)
					d.POST("/network/create", app.dockerCreateNetwork)
					d.GET("/volumes", app.dockerListVolumes)
					d.GET("/volume/:name", app.dockerVolumeInspect)
					d.POST("/volume/action", app.dockerVolumeAction)
					d.POST("/volume/create", app.dockerCreateVolume)
					d.GET("/registries", app.dockerListRegistries)
					d.POST("/registries", app.dockerCreateRegistry)
					d.PUT("/registries", app.dockerUpdateRegistry)
					d.DELETE("/registries", app.dockerDeleteRegistry)
					d.POST("/registry/push", app.dockerPushImage)
					d.POST("/registry/pull", app.dockerPullFromRegistry)
				}
			}

			// Swarm
			sw := auth.Group("/swarm")
			{
				sw.GET("/info", app.swarmInfo)
				sw.GET("/nodes", app.swarmListNodes)
				sw.GET("/node/:id", app.swarmInspectNode)
				sw.POST("/node/action", app.swarmNodeAction)
				sw.GET("/services", app.swarmListServices)
				sw.GET("/service/:id", app.swarmInspectService)
				sw.POST("/service/create", app.swarmCreateService)
				sw.POST("/service/action", app.swarmServiceAction)
				sw.POST("/service/redeploy", app.swarmForceUpdateService)
				sw.GET("/service/:id/logs", app.swarmServiceLogs)
				sw.GET("/tasks", app.swarmListTasks)
			}

			// Compose
			if app.composeSvc != nil {
				auth.POST("/compose/deploy", app.composeDeploy)
			}

			// 系统
			sys := auth.Group("/system")
			{
				sys.GET("/stats", app.systemStat)
				sys.GET("/probe", app.systemProbe)
				sys.GET("/health", app.systemHealth)
				sys.GET("/settings", app.systemGetSettings)
				sys.PUT("/settings", app.systemUpdateSettings)
				sys.GET("/members", app.systemListMembers)
				sys.POST("/members", app.systemCreateMember)
				sys.PUT("/member/:username", app.systemUpdateMember)
				sys.DELETE("/member/:username", app.systemDeleteMember)
			}
		}
	}

	// WebSocket
	ws := app.Group("/ws")
	ws.Use(AuthMiddleware())
	{
		ws.GET("/shell", app.shellWebSocket)
		if app.dockerSvc != nil {
			ws.GET("/docker/container/exec", app.dockerContainerExec)
		}
	}
}
