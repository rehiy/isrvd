package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"isrvd/server/config"
)

func main() {
	// 先解析并校验参数：config.Init 会补全默认值并回写配置文件，
	// 参数错误或 --help 不应产生这个副作用
	opt, err := parse(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "isrvd:", err)
		os.Exit(2)
	}

	if err := config.Init(); err != nil {
		panic(err)
	}
	defer config.Close()

	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, "isrvd:", err)
		config.Close()
		os.Exit(1)
	}
}
