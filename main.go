package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"filer/server"
)

//go:embed public/*
var efs embed.FS

func main() {
	server.InitConfig()
	server.InitRouter()

	// 根路径直接绑定到 /public 目录
	publicFS, err := fs.Sub(efs, "public")
	if err != nil {
		log.Fatal("Failed to create sub filesystem:", err)
	}
	http.Handle("/", http.FileServer(http.FS(publicFS)))

	log.Println("Server started at :8080")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
