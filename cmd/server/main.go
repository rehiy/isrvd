package main

import (
	"isrvd/config"
	"isrvd/internal/registry"
	"isrvd/internal/server"
)

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}

	registry.Init()
	server.StartApp()
}
