package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

var fsetCache = token.NewFileSet() // 复用 FileSet

// ─── 辅助函数 ───

var fileCache = map[string]*ast.File{} // 缓存已解析的文件

// parseFile 解析单个文件，结果缓存
func parseFile(filename string) *ast.File {
	if cached, ok := fileCache[filename]; ok {
		return cached
	}
	f, err := parser.ParseFile(fsetCache, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	fileCache[filename] = f
	return f
}

var contentCache = map[string][]string{} // 文件内容行缓存

// getFileLines 获取文件内容行（带缓存）
func getFileLines(filename string) []string {
	if cached, ok := contentCache[filename]; ok {
		return cached
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")
	contentCache[filename] = lines
	return lines
}
