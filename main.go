package main

import (
	"isrvd/server"
	"isrvd/server/config"
)

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}
	server.Start()
}
