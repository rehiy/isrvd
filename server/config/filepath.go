package config

import (
	"path/filepath"
	"strings"
)

// PathToAbs 将路径转为绝对路径。
// 空值返回 rootDir；相对路径基于 rootDir 拼接；绝对路径原样返回。
func PathToAbs(path string, rootDir string) string {
	if rootDir == "" {
		if path == "" {
			return "."
		}
		return filepath.Clean(path)
	}
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

// ─── 辅助函数 ───

// denormalizePaths 将 conf 中 Server.RootDirectory 内的绝对路径还原为相对路径。
// 注意：此函数会直接修改传入的 conf 对象。
// 只会转换 Server.RootDirectory 内部的绝对路径，外部路径和相对路径保持不变。
func denormalizePaths(conf *Config) {
	if conf == nil || conf.Server == nil {
		return
	}
	if conf.Server.RootDirectory == "" {
		return
	}
	if conf.Docker != nil {
		conf.Docker.ContainerRoot = PathToRel(conf.Docker.ContainerRoot, conf.Server.RootDirectory)
	}
	for _, m := range conf.Members {
		if m != nil {
			m.HomeDirectory = PathToRel(m.HomeDirectory, conf.Server.RootDirectory)
		}
	}
}
