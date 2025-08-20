package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"filer/internal/config"
	"filer/internal/router"

	"github.com/gin-gonic/gin"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	// 初始化配置
	cfg := config.Init()

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 初始化路由
	r := router.SetupRouter()

	// 设置静态文件服务
	efs, _ := fs.Sub(publicFS, "public")
	fileServer := http.FileServer(http.FS(efs))

	r.NoRoute(func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	log.Printf("Server started at :%s", cfg.Port)
	log.Printf("Base directory: %s", cfg.BaseDir)
	log.Printf("Users configured: %d", len(cfg.UserMap))

	r.Run("0.0.0.0:" + cfg.Port)
}
