// Command openapi-gen 从 isrvd 源码自动生成 OpenAPI 3.0 文档，核心逻辑见 server/openapi。
//
// 用法:
//
//	go run ./server/cmd/openapi-gen/  # 生成到 public/openapi/data.json
//	go run ./server/cmd/openapi-gen/ -o openapi.json  # 生成到指定路径
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"isrvd/server/openapi"
)

func main() {
	outFile := flag.String("o", "", "输出文件路径（覆盖默认值）")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatalf("获取工作目录失败: %v", err)
	}

	cfg := openapi.DefaultConfig()
	if *outFile != "" {
		cfg.OutputFile = *outFile
	}

	out, err := openapi.Generate(root, cfg)
	if err != nil {
		fatalf("%v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.OutputFile), 0755); err != nil {
		fatalf("创建输出目录失败: %v", err)
	}
	if err := os.WriteFile(cfg.OutputFile, out, 0644); err != nil {
		fatalf("写入文件失败: %v", err)
	}
	fmt.Fprintf(os.Stderr, "OpenAPI 文档已生成: %s (%d bytes)\n", cfg.OutputFile, len(out))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
