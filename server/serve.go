package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/public"
	"isrvd/server/config"
	"isrvd/server/handler"
	"isrvd/server/middleware"
)

type App struct {
	*gin.Engine
}

func Start() {
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	app := &App{gin.Default()}
	app.create()
}

// 设置路由
func (app *App) create() {
	// 注册中间件
	app.Use(middleware.CORS())

	// 注册模块路由
	app.setupRouter()

	// 静态文件服务
	hfs := http.FileServer(http.FS(public.Efs))
	app.NoRoute(func(c *gin.Context) {
		hfs.ServeHTTP(c.Writer, c.Request)
	})

	log.Printf("Members: %d", len(config.Members))
	log.Printf("Root directory: %s", config.RootDirectory)
	log.Printf("Server started at %s", config.ListenAddr)
	app.Run(config.ListenAddr)
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
		auth.Use(middleware.Auth())
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
	app.GET("/ws/shell", middleware.Auth(), shellHandler.HandleWebSocket)
}
