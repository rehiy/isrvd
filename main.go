package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"filer/server"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	server.InitConfig()
	server.InitRouter()

	// 根路径直接绑定到 /public 目录
	efs, _ := fs.Sub(publicFS, "public")
	http.Handle("/", http.FileServer(http.FS(efs)))

	log.Println("Server started at :8080")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
