package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/public"
	"isrvd/server/config"
	"isrvd/server/handler"
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
		logman.Fatal("Docker client init failed, Docker features disabled", "error", err)
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
			auth.POST("/list", fileHandler.List)
			auth.POST("/upload", fileHandler.Upload)
			auth.POST("/download", fileHandler.Download)
			auth.POST("/delete", fileHandler.Delete)
			auth.POST("/mkdir", fileHandler.Mkdir)
			auth.POST("/create", fileHandler.Create)
			auth.POST("/read", fileHandler.Read)
			auth.POST("/modify", fileHandler.Modify)
			auth.POST("/rename", fileHandler.Rename)
			auth.POST("/chmod", fileHandler.Chmod)
			auth.POST("/zip", zipHandler.Zip)
			auth.POST("/unzip", zipHandler.Unzip)

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
