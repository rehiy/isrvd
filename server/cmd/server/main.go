package main

import (
	"fmt"
	"os"

	"isrvd/server/config"
)

func main() {
	if err := config.Init(); err != nil {
		panic(err)
	}
	defer config.Close()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "isrvd:", err)
		config.Close()
		os.Exit(1)
	}
}
