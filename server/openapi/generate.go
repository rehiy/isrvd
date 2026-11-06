// Package openapi 从 isrvd 源码静态分析并生成 OpenAPI 3.0 文档。
//
// 工作原理:
//  1. 解析 server/app/ctrl_*.go 中的 define*Routes() 方法，提取所有路由定义
//  2. 分析 handler 函数体，提取 ShouldBindJSON/ShouldBindQuery 引用的请求结构体
//  3. 从 service/ 下各业务包中解析结构体定义及其 json tag
//  4. 提取 c.Param / c.Query / c.DefaultQuery 调用，推断路径参数和查询参数
//  5. 组装成标准 OpenAPI 3.0.3 JSON 输出
//
// 每次 Generate 调用使用独立的 generator，无包级可变状态，可并发调用。
package openapi

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
)

// generator 持有一次生成过程中的全部状态
type generator struct {
	root string         // 项目根目录
	cfg  *ProjectConfig // 生成配置

	routes       []RouteDef              // 已解析的路由
	inlineFuncs  map[string]*ast.FuncLit // 内联 handler 闭包，键为 inline:<id>
	inlineSeq    int                     // 内联 handler 序号
	structCache  map[string]*SchemaInfo  // "pkg.TypeName" -> schema
	ctrlEntries  []os.DirEntry           // ctrl 目录条目缓存
	fset         *token.FileSet          // 复用 FileSet
	fileCache    map[string]*ast.File    // 已解析文件缓存
	contentCache map[string][]string     // 文件内容行缓存
}

func newGenerator(root string, cfg *ProjectConfig) *generator {
	return &generator{
		root:         root,
		cfg:          cfg,
		inlineFuncs:  map[string]*ast.FuncLit{},
		structCache:  map[string]*SchemaInfo{},
		fset:         token.NewFileSet(),
		fileCache:    map[string]*ast.File{},
		contentCache: map[string][]string{},
	}
}

// Generate 以 root 为项目根目录，按 cfg 的配置生成 OpenAPI JSON。cfg 为 nil 时使用 DefaultConfig。
func Generate(root string, cfg *ProjectConfig) ([]byte, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return newGenerator(root, cfg).run()
}

func (g *generator) run() ([]byte, error) {
	// 0. 预注册 APIResponse schema（从源文件提取字段注释）
	responseFile := filepath.Join(g.root, g.cfg.ResponseFile)
	if schema := g.parseServiceStruct("", g.cfg.ResponseStruct, responseFile); schema != nil {
		g.structCache[g.cfg.ResponseStruct] = schema
	} else {
		// 降级处理：如果无法从文件提取，使用默认值
		g.structCache[g.cfg.ResponseStruct] = &SchemaInfo{
			TypeName: g.cfg.ResponseStruct,
			Fields: []FieldInfo{
				{Name: "success", Type: "bool", Required: true, Description: "请求是否成功"},
				{Name: "message", Type: "string", Description: "提示信息"},
				{Name: "payload", Type: "any", Description: "响应数据负载"},
			},
		}
	}

	// 1-2. 解析 ctrl 文件 & 预收集类型信息
	ctrlDir := filepath.Join(g.root, g.cfg.CtrlDir)
	entries, err := os.ReadDir(ctrlDir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, filename := range g.getCtrlFiles(ctrlDir, entries) {
		g.parseCtrlFile(filename)
		g.collectAllTypesFromFile(filename)
	}

	// 2.5 解析 service 及额外类型目录中的类型定义
	g.collectServiceTypes()

	// 3. 对每个路由的 handler 分析请求类型
	for i := range g.routes {
		g.analyzeHandler(&g.routes[i])
	}

	// 4. 生成 OpenAPI JSON
	return json.MarshalIndent(g.buildOpenAPI(), "", "  ")
}
