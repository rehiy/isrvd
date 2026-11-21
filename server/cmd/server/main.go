package main

import (
	"fmt"
	"os"

	"isrvd/server/config"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--portmap-range" {
		portRange, err := config.PortMapRangeFromFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "isrvd:", err)
			os.Exit(1)
		}
		fmt.Println(portRange)
		return
	}
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
