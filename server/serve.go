package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/public"
	"isrvd/server/config"
	apisixHandler "isrvd/server/handler/apisix"
	authHandler "isrvd/server/handler/auth"
	dockerHandler "isrvd/server/handler/docker"
	filerHandler "isrvd/server/handler/filer"
	shellHandler "isrvd/server/handler/shell"
	swarmHandler "isrvd/server/handler/swarm"
	"isrvd/server/middleware"
)

type App struct {
	*gin.Engine
}

func Start() {
	app := &App{httpd.Engine(config.Debug)}
	app.create()
}

// 设置路由
func (app *App) create() {
	// 注册中间件
	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.RecoveryMiddleware())

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
}

// 设置管理器路由
func (app *App) setupRouter() {
	ah := authHandler.NewAuthHandler()
	fh := filerHandler.NewFileHandler()
	sh := shellHandler.NewShellHandler()
	zh := filerHandler.NewZipHandler()

	// 注册 Docker Handler
	dh, err := dockerHandler.NewDockerHandler()
	if err != nil {
		logman.Error("Docker client init failed", "error", err)
	}

	// 注册 Swarm Handler（复用 Docker client）
	swh := swarmHandler.NewSwarmHandler(dh.GetClient())

	// 注册 Apisix Handler（可选，配置了才启用）
	apih, err := apisixHandler.NewHandler()
	if err != nil {
		logman.Error("Apisix client init failed", "error", err)
	}

	// API 路由组
	api := app.Group("/api")
	{
		// 无需认证的路由
		api.POST("/login", ah.Login)

		// 需认证的路由组
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.POST("/logout", ah.Logout)

			// 文件管理 API 路由
			filer := auth.Group("/filer")
			{
				filer.POST("/list", fh.List)
				filer.POST("/upload", fh.Upload)
				filer.POST("/download", fh.Download)
				filer.POST("/delete", fh.Delete)
				filer.POST("/mkdir", fh.Mkdir)
				filer.POST("/create", fh.Create)
				filer.POST("/read", fh.Read)
				filer.POST("/modify", fh.Modify)
				filer.POST("/rename", fh.Rename)
				filer.POST("/chmod", fh.Chmod)
				filer.POST("/zip", zh.Zip)
				filer.POST("/unzip", zh.Unzip)
			}

			// Apisix API 路由
			apisix := auth.Group("/apisix")
			{
				// 路由管理
				apisix.GET("/routes", apih.ListRoutes)
				apisix.GET("/routes/:id", apih.GetRoute)
				apisix.POST("/routes", apih.CreateRoute)
				apisix.PUT("/routes/:id", apih.UpdateRoute)
				apisix.PATCH("/routes/:id/status", apih.PatchRouteStatus)
				apisix.DELETE("/routes/:id", apih.DeleteRoute)

				// Consumer 管理
				apisix.GET("/consumers", apih.ListConsumers)
				apisix.POST("/consumers", apih.CreateConsumer)
				apisix.PUT("/consumers/:username", apih.UpdateConsumer)
				apisix.DELETE("/consumers/:username", apih.DeleteConsumer)

				// 白名单管理
				apisix.GET("/whitelist", apih.GetWhitelist)
				apisix.PUT("/whitelist/revoke", apih.RevokeWhitelist)

				// 辅助资源
				apisix.GET("/plugin_configs", apih.ListPluginConfigs)
				apisix.GET("/upstreams", apih.ListUpstreams)
				apisix.GET("/plugins", apih.ListPlugins)
			}

			// Docker API 路由
			docker := auth.Group("/docker")
			{
				// 概览
				docker.GET("/info", dh.Info)

				// 容器管理
				docker.GET("/containers", dh.ListContainers)
				docker.POST("/container/action", dh.ContainerAction)
				docker.POST("/container/create", dh.CreateContainer)
				docker.POST("/container/logs", dh.ContainerLogs)
				docker.GET("/container/stats", dh.ContainerStats)
				docker.GET("/container/config", dh.GetContainerConfig)
				docker.POST("/container/update", dh.UpdateContainerConfig)

				// 镜像管理
				docker.GET("/images", dh.ListImages)
				docker.POST("/image/action", dh.ImageAction)
				docker.POST("/image/pull", dh.PullImage)
				docker.POST("/image/tag", dh.ImageTag)
				docker.GET("/image/search", dh.ImageSearch)
				docker.POST("/image/build", dh.ImageBuild)

				// 网络管理
				docker.GET("/networks", dh.ListNetworks)
				docker.GET("/network/inspect", dh.NetworkInspect)
				docker.POST("/network/action", dh.NetworkAction)
				docker.POST("/network/create", dh.CreateNetwork)

				// 卷管理
				docker.GET("/volumes", dh.ListVolumes)
				docker.GET("/volume/inspect", dh.VolumeInspect)
				docker.POST("/volume/action", dh.VolumeAction)
				docker.POST("/volume/create", dh.CreateVolume)

				// 镜像仓库管理
				docker.GET("/registries", dh.ListRegistries)
				docker.POST("/registry/push", dh.PushImage)
				docker.POST("/registry/pull", dh.PullFromRegistry)

			}

			// Swarm API 路由
			swarmGroup := auth.Group("/swarm")
			{
				swarmGroup.GET("/info", swh.SwarmInfo)
				swarmGroup.GET("/nodes", swh.SwarmListNodes)
				swarmGroup.POST("/node/action", swh.SwarmNodeAction)
				swarmGroup.GET("/services", swh.SwarmListServices)
				swarmGroup.POST("/service/create", swh.SwarmCreateService)
				swarmGroup.POST("/service/action", swh.SwarmServiceAction)
				swarmGroup.POST("/service/redeploy", swh.SwarmForceUpdateService)
				swarmGroup.GET("/service/logs", swh.SwarmServiceLogs)
				swarmGroup.GET("/tasks", swh.SwarmListTasks)
			}
		}
	}

	// WebSocket 路由
	ws := app.Group("/ws")
	ws.Use(middleware.AuthMiddleware())
	{
		ws.GET("/shell", sh.WebSocket)
		ws.GET("/docker/container/exec", dh.ContainerExec)
	}
}
