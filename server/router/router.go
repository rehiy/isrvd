package router

import (
	"github.com/gin-gonic/gin"

	"isrvd/server/handlers"
	"isrvd/server/middleware"
)

// 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 设置 CORS 中间件
	r.Use(middleware.CORS())

	// 创建处理器实例
	authHandler := handlers.NewAuthHandler()
	fileHandler := handlers.NewFileHandler()
	shellHandler := handlers.NewShellHandler()
	zipHandler := handlers.NewZipHandler()

	// API 路由组
	api := r.Group("/api")
	{
		// 公开路由
		api.POST("/login", authHandler.Login)

		// 需要认证的路由组
		auth := api.Group("")
		auth.Use(middleware.Auth())
		{
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/files", fileHandler.ListFiles)
			auth.POST("/upload", fileHandler.Upload)
			auth.POST("/download", fileHandler.Download)
			auth.POST("/delete", fileHandler.Delete)
			auth.POST("/mkdir", fileHandler.CreateDirectory)
			auth.POST("/newfile", fileHandler.CreateFile)
			auth.POST("/cat", fileHandler.ReadFile)
			auth.POST("/edit", fileHandler.WriteFile)
			auth.POST("/rename", fileHandler.Rename)
			auth.POST("/chmod", fileHandler.ChangeMode)
			auth.POST("/zip", zipHandler.CreateZip)
			auth.POST("/unzip", zipHandler.ExtractZip)
			auth.POST("/zip/info", zipHandler.GetZipInfo)
			auth.POST("/zip/check", zipHandler.IsZipFile)
		}
	}

	// WebSocket 路由
	r.GET("/ws/shell", shellHandler.HandleWebSocket)

	return r
}
