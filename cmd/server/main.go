package main

import (
	"isrvd/config"
	"isrvd/server"
)

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}
	server.Start()
}
