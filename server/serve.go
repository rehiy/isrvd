package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/public"
	"isrvd/server/config"
	"isrvd/server/handler"
	"isrvd/server/handler/apisix"
	"isrvd/server/handler/docker"
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
	authHandler := handler.NewAuthHandler()
	fileHandler := handler.NewFileHandler()
	shellHandler := handler.NewShellHandler()
	zipHandler := handler.NewZipHandler()

	// 注册 Docker Handler
	dockerHandler, err := docker.NewDockerHandler()
	if err != nil {
		logman.Error("Docker client init failed", "error", err)
	}

	// 注册 Apisix Handler（可选，配置了才启用）
	apisixHandler, err := apisix.NewHandler()
	if err != nil {
		logman.Error("Apisix client init failed", "error", err)
	}

	// API 路由组
	api := app.Group("/api")
	{
		// 无需认证的路由
		api.POST("/login", authHandler.Login)

		// 需认证的路由组
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.POST("/logout", authHandler.Logout)

			// 文件管理 API 路由
			filer := auth.Group("/filer")
			{
				filer.POST("/list", fileHandler.List)
				filer.POST("/upload", fileHandler.Upload)
				filer.POST("/download", fileHandler.Download)
				filer.POST("/delete", fileHandler.Delete)
				filer.POST("/mkdir", fileHandler.Mkdir)
				filer.POST("/create", fileHandler.Create)
				filer.POST("/read", fileHandler.Read)
				filer.POST("/modify", fileHandler.Modify)
				filer.POST("/rename", fileHandler.Rename)
				filer.POST("/chmod", fileHandler.Chmod)
				filer.POST("/zip", zipHandler.Zip)
				filer.POST("/unzip", zipHandler.Unzip)
			}

			// Apisix API 路由
			apisix := auth.Group("/apisix")
			{
				// 路由管理
				apisix.GET("/routes", apisixHandler.ListRoutes)
				apisix.GET("/routes/:id", apisixHandler.GetRoute)
				apisix.POST("/routes", apisixHandler.CreateRoute)
				apisix.PUT("/routes/:id", apisixHandler.UpdateRoute)
				apisix.PATCH("/routes/:id/status", apisixHandler.PatchRouteStatus)
				apisix.DELETE("/routes/:id", apisixHandler.DeleteRoute)

				// Consumer 管理
				apisix.GET("/consumers", apisixHandler.ListConsumers)
				apisix.POST("/consumers", apisixHandler.CreateConsumer)
				apisix.PUT("/consumers/:username", apisixHandler.UpdateConsumer)
				apisix.DELETE("/consumers/:username", apisixHandler.DeleteConsumer)

				// 白名单管理
				apisix.GET("/whitelist", apisixHandler.GetWhitelist)
				apisix.PUT("/whitelist/revoke", apisixHandler.RevokeWhitelist)

				// 辅助资源
				apisix.GET("/plugin_configs", apisixHandler.ListPluginConfigs)
				apisix.GET("/upstreams", apisixHandler.ListUpstreams)
				apisix.GET("/plugins", apisixHandler.ListPlugins)
			}

			// Docker API 路由
			docker := auth.Group("/docker")
			{
				// 概览
				docker.GET("/info", dockerHandler.Info)

				// 容器管理
				docker.GET("/containers", dockerHandler.ListContainers)
				docker.POST("/container/action", dockerHandler.ContainerAction)
				docker.POST("/container/create", dockerHandler.CreateContainer)
				docker.POST("/container/logs", dockerHandler.ContainerLogs)
				docker.GET("/container/stats", dockerHandler.ContainerStats)
				docker.GET("/container/config", dockerHandler.GetContainerConfig)
				docker.POST("/container/update", dockerHandler.UpdateContainerConfig)

				// 镜像管理
				docker.GET("/images", dockerHandler.ListImages)
				docker.POST("/image/action", dockerHandler.ImageAction)
				docker.POST("/image/pull", dockerHandler.PullImage)
				docker.POST("/image/tag", dockerHandler.ImageTag)
				docker.GET("/image/search", dockerHandler.ImageSearch)
				docker.POST("/image/build", dockerHandler.ImageBuild)

				// 网络管理
				docker.GET("/networks", dockerHandler.ListNetworks)
				docker.GET("/network/inspect", dockerHandler.NetworkInspect)
				docker.POST("/network/action", dockerHandler.NetworkAction)
				docker.POST("/network/create", dockerHandler.CreateNetwork)

				// 卷管理
				docker.GET("/volumes", dockerHandler.ListVolumes)
				docker.GET("/volume/inspect", dockerHandler.VolumeInspect)
				docker.POST("/volume/action", dockerHandler.VolumeAction)
				docker.POST("/volume/create", dockerHandler.CreateVolume)

				// 镜像仓库管理
				docker.GET("/registries", dockerHandler.ListRegistries)
				docker.POST("/registry/push", dockerHandler.PushImage)
				docker.POST("/registry/pull", dockerHandler.PullFromRegistry)

				// Swarm 集群管理
				docker.GET("/swarm/info", dockerHandler.SwarmInfo)
				docker.GET("/swarm/nodes", dockerHandler.SwarmListNodes)
				docker.POST("/swarm/node/action", dockerHandler.SwarmNodeAction)
				docker.GET("/swarm/services", dockerHandler.SwarmListServices)
				docker.POST("/swarm/service/action", dockerHandler.SwarmServiceAction)
				docker.GET("/swarm/tasks", dockerHandler.SwarmListTasks)
			}
		}
	}

	// WebSocket 路由
	ws := app.Group("/ws")
	ws.Use(middleware.AuthMiddleware())
	{
		ws.GET("/shell", shellHandler.WebSocket)
		ws.GET("/docker/container/exec", dockerHandler.ContainerExec)
	}
}
