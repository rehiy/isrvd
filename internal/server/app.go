package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/handler/apisix"
	"isrvd/internal/handler/auth"
	"isrvd/internal/handler/docker"
	"isrvd/internal/handler/filer"
	"isrvd/internal/handler/settings"
	"isrvd/internal/handler/shell"
	"isrvd/internal/handler/swarm"
	"isrvd/internal/handler/system"
	"isrvd/public"
)

type App struct {
	*gin.Engine
}

func NewApp() *App {
	app := &App{httpd.Engine(config.Debug)}

	// 注册模块路由
	app.setupRouter()

	// 输出服务器信息
	logman.Info("Server starting",
		"listenAddr", config.ListenAddr,
		"rootDirectory", config.RootDirectory,
		"members", len(config.Members),
	)

	httpd.StaticEmbed(public.Efs, "", "")
	httpd.Server(config.ListenAddr)

	return app
}

// 设置管理器路由
func (app *App) setupRouter() {
	authHandler := auth.NewAuthHandler()
	fileHandler := filer.NewFileHandler()
	zipHandler := filer.NewZipHandler()
	shellHandler := shell.NewShellHandler()
	settingsHandler := settings.NewSettingsHandler()
	systemHandler := system.NewSystemHandler()

	// 注册 Apisix Handler
	apisixHandler, _ := apisix.NewHandler()

	// 注册 Docker Handler
	dockerHandler, _ := docker.NewDockerHandler()

	// 注册 Swarm Handler
	swarmHandler := swarm.NewSwarmHandler()

	// API 路由组
	api := app.Group("/api")
	{
		// 无需认证的路由
		api.POST("/login", authHandler.Login)

		// 需认证的路由组
		authGroup := api.Group("")
		authGroup.Use(AuthMiddleware())
		{
			authGroup.POST("/logout", authHandler.Logout)

			// 文件管理 API 路由
			filerGroup := authGroup.Group("/filer")
			{
				filerGroup.POST("/list", fileHandler.List)
				filerGroup.POST("/upload", fileHandler.Upload)
				filerGroup.POST("/download", fileHandler.Download)
				filerGroup.POST("/delete", fileHandler.Delete)
				filerGroup.POST("/mkdir", fileHandler.Mkdir)
				filerGroup.POST("/create", fileHandler.Create)
				filerGroup.POST("/read", fileHandler.Read)
				filerGroup.POST("/modify", fileHandler.Modify)
				filerGroup.POST("/rename", fileHandler.Rename)
				filerGroup.POST("/chmod", fileHandler.Chmod)
				filerGroup.POST("/zip", zipHandler.Zip)
				filerGroup.POST("/unzip", zipHandler.Unzip)
			}

			// Apisix API 路由
			if apisixHandler != nil {
				apisixGroup := authGroup.Group("/apisix")
				{
					// 路由管理
					apisixGroup.GET("/routes", apisixHandler.ListRoutes)
					apisixGroup.GET("/routes/:id", apisixHandler.GetRoute)
					apisixGroup.POST("/routes", apisixHandler.CreateRoute)
					apisixGroup.PUT("/routes/:id", apisixHandler.UpdateRoute)
					apisixGroup.PATCH("/routes/:id/status", apisixHandler.PatchRouteStatus)
					apisixGroup.DELETE("/routes/:id", apisixHandler.DeleteRoute)

					// Consumer 管理
					apisixGroup.GET("/consumers", apisixHandler.ListConsumers)
					apisixGroup.POST("/consumers", apisixHandler.CreateConsumer)
					apisixGroup.PUT("/consumers/:username", apisixHandler.UpdateConsumer)
					apisixGroup.DELETE("/consumers/:username", apisixHandler.DeleteConsumer)

					// 插件管理
					apisixGroup.GET("/plugin_configs", apisixHandler.ListPluginConfigs)
					apisixGroup.GET("/plugins", apisixHandler.ListPlugins)

					// 上游管理
					apisixGroup.GET("/upstreams", apisixHandler.ListUpstreams)

					// 白名单管理
					apisixGroup.GET("/whitelist", apisixHandler.GetWhitelist)
					apisixGroup.PUT("/whitelist/revoke", apisixHandler.RevokeWhitelist)
				}
			}

			// Docker API 路由
			if dockerHandler != nil {
				dockerGroup := authGroup.Group("/docker")
				{
					// 概览
					dockerGroup.GET("/info", dockerHandler.Info)

					// 容器管理
					dockerGroup.GET("/containers", dockerHandler.ListContainers)
					dockerGroup.POST("/container/action", dockerHandler.ContainerAction)
					dockerGroup.POST("/container/create", dockerHandler.CreateContainer)
					dockerGroup.POST("/container/deploy-compose", dockerHandler.DeployCompose)
					dockerGroup.POST("/container/logs", dockerHandler.ContainerLogs)
					dockerGroup.GET("/container/stats", dockerHandler.ContainerStats)
					dockerGroup.GET("/container/config", dockerHandler.GetContainerConfig)
					dockerGroup.POST("/container/update", dockerHandler.UpdateContainerConfig)

					// 镜像管理
					dockerGroup.GET("/images", dockerHandler.ListImages)
					dockerGroup.POST("/image/action", dockerHandler.ImageAction)
					dockerGroup.GET("/image/inspect", dockerHandler.InspectImage)
					dockerGroup.POST("/image/pull", dockerHandler.PullImage)
					dockerGroup.POST("/image/tag", dockerHandler.TagImage)
					dockerGroup.GET("/image/search", dockerHandler.SearchImages)
					dockerGroup.POST("/image/build", dockerHandler.BuildImage)

					// 网络管理
					dockerGroup.GET("/networks", dockerHandler.ListNetworks)
					dockerGroup.GET("/network/inspect", dockerHandler.NetworkInspect)
					dockerGroup.POST("/network/action", dockerHandler.NetworkAction)
					dockerGroup.POST("/network/create", dockerHandler.CreateNetwork)

					// 卷管理
					dockerGroup.GET("/volumes", dockerHandler.ListVolumes)
					dockerGroup.GET("/volume/inspect", dockerHandler.VolumeInspect)
					dockerGroup.POST("/volume/action", dockerHandler.VolumeAction)
					dockerGroup.POST("/volume/create", dockerHandler.CreateVolume)

					// 镜像仓库管理
					dockerGroup.GET("/registries", dockerHandler.ListRegistries)
					dockerGroup.POST("/registries", dockerHandler.CreateRegistry)
					dockerGroup.PUT("/registries", dockerHandler.UpdateRegistry)
					dockerGroup.DELETE("/registries", dockerHandler.DeleteRegistry)
					dockerGroup.POST("/registry/push", dockerHandler.PushImage)
					dockerGroup.POST("/registry/pull", dockerHandler.PullFromRegistry)
				}
			}

			// Swarm API 路由
			if swarmHandler != nil {
				swarmGroup := authGroup.Group("/swarm")
				{
					// 概览
					swarmGroup.GET("/info", swarmHandler.SwarmInfo)

					// 节点管理
					swarmGroup.GET("/nodes", swarmHandler.SwarmListNodes)
					swarmGroup.GET("/node/inspect", swarmHandler.SwarmInspectNode)
					swarmGroup.POST("/node/action", swarmHandler.SwarmNodeAction)

					// 服务管理
					swarmGroup.GET("/services", swarmHandler.SwarmListServices)
					swarmGroup.GET("/service/inspect", swarmHandler.SwarmInspectService)
					swarmGroup.POST("/service/create", swarmHandler.SwarmCreateService)
					swarmGroup.POST("/service/deploy-compose", swarmHandler.SwarmDeployComposeService)
					swarmGroup.POST("/service/action", swarmHandler.SwarmServiceAction)
					swarmGroup.POST("/service/redeploy", swarmHandler.SwarmForceUpdateService)
					swarmGroup.GET("/service/logs", swarmHandler.SwarmServiceLogs)

					// 任务管理
					swarmGroup.GET("/tasks", swarmHandler.SwarmListTasks)
				}
			}

			// 系统信息 API 路由
			systemGroup := authGroup.Group("/system")
			{
				systemGroup.GET("/stat", systemHandler.Stat)
				systemGroup.GET("/probe", systemHandler.Probe)
				systemGroup.GET("/health", systemHandler.Health)
			}

			// 系统设置 API 路由
			settingsGroup := authGroup.Group("/settings")
			{
				settingsGroup.GET("", settingsHandler.GetAll)
				settingsGroup.PUT("/server", settingsHandler.UpdateServer)
				settingsGroup.PUT("/apisix", settingsHandler.UpdateApisix)
				settingsGroup.PUT("/docker", settingsHandler.UpdateDocker)

				// 成员账号
				settingsGroup.GET("/members", settingsHandler.ListMembers)
				settingsGroup.POST("/members", settingsHandler.CreateMember)
				settingsGroup.PUT("/members/:username", settingsHandler.UpdateMember)
				settingsGroup.DELETE("/members/:username", settingsHandler.DeleteMember)
			}
		}
	}

	// WebSocket 路由
	ws := app.Group("/ws")
	ws.Use(AuthMiddleware())
	{
		ws.GET("/shell", shellHandler.WebSocket)
		if dockerHandler != nil {
			ws.GET("/docker/container/exec", dockerHandler.ContainerExec)
		}
	}
}
