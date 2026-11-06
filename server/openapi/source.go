package openapi

import (
	"go/ast"
	"go/parser"
	"os"
	"strings"
)

// ─── 辅助函数 ───

// parseFile 解析单个文件，结果缓存
func (g *generator) parseFile(filename string) *ast.File {
	if cached, ok := g.fileCache[filename]; ok {
		return cached
	}
	f, err := parser.ParseFile(g.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	g.fileCache[filename] = f
	return f
}

// getFileLines 获取文件内容行（带缓存）
func (g *generator) getFileLines(filename string) []string {
	if cached, ok := g.contentCache[filename]; ok {
		return cached
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")
	g.contentCache[filename] = lines
	return lines
}
