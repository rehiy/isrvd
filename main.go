package main

import (
	"embed"

	"isrvd/server"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	server.StartHTTP(publicFS)
}
