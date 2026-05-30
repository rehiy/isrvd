package config

import (
	"path/filepath"
	"strings"
)

// PathToAbs 将路径转为绝对路径。
// 空值返回 rootDir；相对路径基于 rootDir 拼接；绝对路径原样返回。
func PathToAbs(rootDir, path string) string {
	if path == "" {
		return filepath.Clean(rootDir)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(rootDir, path)
	}
	return filepath.Clean(path)
}

// PathToRel 将绝对路径转为基于 rootDir 的相对路径（"./" 前缀），
// 仅在 path 位于 rootDir 内部时转换，否则返回原绝对路径。
func PathToRel(rootDir, path string) string {
	if path == "" || !filepath.IsAbs(path) {
		return path
	}
	root := filepath.Clean(rootDir)
	target := filepath.Clean(path)
	// 获取真实路径（解析符号链接）
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	if t, err := filepath.EvalSymlinks(target); err == nil {
		target = t
	}
	// 检查 target 是否在 root 内部：真实路径以 root 开头即在内部
	rootWithSep := root + string(filepath.Separator)
	if target != root && !strings.HasPrefix(target, rootWithSep) {
		return path
	}
	// 计算相对路径
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return path
	}
	if rel == "." {
		return "." + string(filepath.Separator)
	}
	return "." + string(filepath.Separator) + rel
}

// denormalizePaths 将 conf 中 rootDir 内的绝对路径还原为相对路径
func denormalizePaths(conf *Config) {
	if conf == nil || conf.Server == nil {
		return
	}
	root := conf.Server.RootDirectory
	if root == "" {
		return
	}
	if conf.Docker != nil {
		conf.Docker.ContainerRoot = PathToRel(root, conf.Docker.ContainerRoot)
	}
	for _, m := range conf.Members {
		if m != nil {
			m.HomeDirectory = PathToRel(root, m.HomeDirectory)
		}
	}
}
