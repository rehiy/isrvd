package main

import (
	"isrvd/server/config"
	"isrvd/server/app"
)

func main() {
	if err := config.Init(); err != nil {
		panic(err)
	}
	defer config.Close()

	app.StartApp()
}
