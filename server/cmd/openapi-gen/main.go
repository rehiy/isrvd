// Command openapi-gen 从 isrvd 源码自动生成 OpenAPI 3.0 文档。
//
// 用法:
//
//	go run ./server/cmd/openapi-gen/  # 生成到 public/openapi/data.json
//	go run ./server/cmd/openapi-gen/ -o openapi.json  # 生成到指定路径
//
// 工作原理:
//  1. 解析 server/app/ctrl_*.go 中的 define*Routes() 方法，提取所有路由定义
//  2. 分析 handler 函数体，提取 ShouldBindJSON/ShouldBindQuery 引用的请求结构体
//  3. 从 service/ 下各业务包中解析结构体定义及其 json tag
//  4. 提取 c.Param / c.Query / c.DefaultQuery 调用，推断路径参数和查询参数
//  5. 组装成标准 OpenAPI 3.0.3 JSON 输出
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	projectRoot string
	cfg         *ProjectConfig
)

// outFile 输出文件路径（覆盖默认值）
var outFile = flag.String("o", "", "输出文件路径（覆盖默认值）")

func main() {
	flag.Parse()
	var err error
	projectRoot, err = os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化配置
	cfg = DefaultConfig()
	if *outFile != "" {
		cfg.OutputFile = *outFile
	}

	// 0. 预注册 APIResponse schema（从源文件提取字段注释）
	responseFile := filepath.Join(projectRoot, cfg.ResponseFile)
	if schema := parseServiceStruct("", cfg.ResponseStruct, responseFile); schema != nil {
		structCache[cfg.ResponseStruct] = schema
	} else {
		// 降级处理：如果无法从文件提取，使用默认值
		structCache[cfg.ResponseStruct] = &SchemaInfo{
			TypeName: cfg.ResponseStruct,
			Fields: []FieldInfo{
				{Name: "success", Type: "bool", Required: true, Description: "请求是否成功"},
				{Name: "message", Type: "string", Description: "提示信息"},
				{Name: "payload", Type: "any", Description: "响应数据负载"},
			},
		}
	}

	// 1-2. 解析 ctrl 文件 & 预收集类型信息
	ctrlDir := filepath.Join(projectRoot, cfg.CtrlDir)
	entries, err := os.ReadDir(ctrlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取目录失败: %v\n", err)
		os.Exit(1)
	}

	ctrlFiles := getCtrlFiles(ctrlDir, entries)
	for _, filename := range ctrlFiles {
		parseCtrlFile(filename)
		collectAllTypesFromFile(filename)
	}

	// 2.5 解析 service 及额外类型目录中的类型定义
	collectServiceTypes()

	// 3. 对每个路由的 handler 分析请求类型
	for i := range routes {
		analyzeHandler(&routes[i])
	}

	// 4. 生成 OpenAPI JSON
	openapi := buildOpenAPI()
	out, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化失败: %v\n", err)
		os.Exit(1)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(cfg.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(cfg.OutputFile, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "OpenAPI 文档已生成: %s (%d bytes)\n", cfg.OutputFile, len(out))
}
