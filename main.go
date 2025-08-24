package main

import (
	"embed"

	"isrvd/server"
	"isrvd/server/config"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	config.PublicFS = publicFS
	server.StartHTTP()
}
