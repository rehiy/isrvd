package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/httpd"
	"github.com/rehiy/pango/logman"

	"isrvd/public"
	"isrvd/server/config"
	"isrvd/server/handler"
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

	dockerHandler, err := handler.NewDockerHandler()
	if err != nil {
		logman.Warn("Docker client init failed, Docker features disabled", "error", err)
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
			if dockerHandler != nil {
				docker := auth.Group("/docker")
				{
					// 概览
					docker.GET("/info", dockerHandler.Info)

					// 容器管理
					docker.GET("/containers", dockerHandler.ListContainers)
					docker.POST("/containers/action", dockerHandler.ContainerAction)
					docker.POST("/containers/create", dockerHandler.CreateContainer)
					docker.POST("/containers/logs", dockerHandler.ContainerLogs)

					// 镜像管理
					docker.GET("/images", dockerHandler.ListImages)
					docker.POST("/images/action", dockerHandler.ImageAction)
					docker.POST("/images/pull", dockerHandler.PullImage)

					// 网络管理
					docker.GET("/networks", dockerHandler.ListNetworks)
					docker.POST("/networks/action", dockerHandler.NetworkAction)
					docker.POST("/networks/create", dockerHandler.CreateNetwork)

					// 卷管理
					docker.GET("/volumes", dockerHandler.ListVolumes)
					docker.POST("/volumes/action", dockerHandler.VolumeAction)
					docker.POST("/volumes/create", dockerHandler.CreateVolume)
				}
			}
		}
	}

	// WebSocket 路由
	app.GET("/ws/shell", middleware.AuthMiddleware(), shellHandler.WebSocket)
}
