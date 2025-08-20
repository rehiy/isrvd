package router

import (
	"filer/internal/handlers"
	"filer/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
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
			// 认证相关
			auth.POST("/logout", authHandler.Logout)

			// 文件操作
			auth.GET("/files", fileHandler.ListFiles)
			auth.POST("/upload", fileHandler.Upload)
			auth.GET("/download", fileHandler.Download)
			auth.DELETE("/delete", fileHandler.Delete)
			auth.POST("/mkdir", fileHandler.CreateDirectory)
			auth.POST("/newfile", fileHandler.CreateFile)
			auth.PUT("/edit", fileHandler.EditFile)
			auth.GET("/edit", fileHandler.EditFile) // 支持GET读取文件内容
			auth.PUT("/rename", fileHandler.Rename)
			auth.PUT("/chmod", fileHandler.ChangeMode)
			auth.GET("/chmod", fileHandler.ChangeMode) // 支持GET获取当前权限

			// Zip 操作
			auth.POST("/zip", zipHandler.CreateZip)
			auth.POST("/unzip", zipHandler.ExtractZip)
			auth.GET("/zip/info", zipHandler.GetZipInfo)
			auth.GET("/zip/check", zipHandler.IsZipFile)
		}
	}

	// WebSocket 路由
	r.GET("/ws/shell", shellHandler.HandleWebSocket)

	return r
}
