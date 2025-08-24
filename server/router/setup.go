package router

import (
	"github.com/gin-gonic/gin"

	"isrvd/server/handler"
	"isrvd/server/middleware"
)

// 设置路由
func Setup() *gin.Engine {
	r := gin.Default()

	// 设置 CORS 中间件
	r.Use(middleware.CORS())

	// 创建处理器实例
	authHandler := handler.NewAuthHandler()
	fileHandler := handler.NewFileHandler()
	shellHandler := handler.NewShellHandler()
	zipHandler := handler.NewZipHandler()

	// API 路由组
	api := r.Group("/api")
	{
		// 公开路由
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
	r.GET("/ws/shell", shellHandler.HandleWebSocket)

	return r
}
