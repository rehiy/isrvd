package config

import (
	"path/filepath"
	"strings"
)

// PathToAbs 基于 rootDir 拼接相对路径并清理结果；rootDir 为绝对路径时结果为绝对路径。
// 空 path 使用 rootDir；两者均为空时返回 "."。
func PathToAbs(path, rootDir string) string {
	if path == "" {
		path = rootDir
	} else if rootDir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(rootDir, path)
	}
	return filepath.Clean(path)
}

// PathToRel 将绝对路径转为基于 rootDir 的相对路径（"./" 前缀），
// 仅在 path 位于 rootDir 内部时转换，否则返回原绝对路径。
func PathToRel(path string, rootDir string) string {
	if rootDir == "" || path == "" || !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(filepath.Clean(rootDir), filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	if rel == "." {
		return "." + string(filepath.Separator)
	}
	return "." + string(filepath.Separator) + rel
}
