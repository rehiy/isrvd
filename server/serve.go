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
		"members", len(config.Members),
		"rootDirectory", config.RootDirectory,
		"listenAddr", config.ListenAddr,
	)

	httpd.StaticEmbed(public.Efs, "", "")
	httpd.Server(config.ListenAddr, config.Debug)
}

// 设置管理器路由
func (app *App) setupRouter() {
	authHandler := handler.NewAuthHandler()
	fileHandler := handler.NewFileHandler()
	shellHandler := handler.NewShellHandler()
	zipHandler := handler.NewZipHandler()

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
		}
	}

	// WebSocket 路由
	app.GET("/ws/shell", middleware.AuthMiddleware(), shellHandler.WebSocket)
}
