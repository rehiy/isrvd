package server

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/config"
	"isrvd/server/router"
)

func StartHTTP() {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 初始化路由
	r := router.Setup()

	// 设置静态文件服务
	efs, _ := fs.Sub(config.PublicFS, "public")
	fileServer := http.FileServer(http.FS(efs))

	r.NoRoute(func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	log.Printf("Base directory: %s", config.BaseDir)
	log.Printf("Users configured: %d", len(config.UserMap))
	log.Printf("Server started at %s", config.Addr)

	r.Run(config.Addr)
}
