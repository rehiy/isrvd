package main

import (
	"isrvd/server/app"
	"isrvd/server/config"
)

func main() {
	if err := config.Init(); err != nil {
		panic(err)
	}
	defer config.Close()

	app.StartApp()
}
