package config

import (
	"os"
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
	if rootDir == "" {
		return path
	}

	if path == "" || !filepath.IsAbs(path) {
		return path
	}
	rootDirClean := filepath.Clean(rootDir)
	target := filepath.Clean(path)

	// 尝试获取真实路径（解析符号链接）
	rootResolved := false
	if r, err := filepath.EvalSymlinks(rootDirClean); err == nil {
		rootDirClean = r
		rootResolved = true
	}
	targetResolved := false
	if t, err := filepath.EvalSymlinks(target); err == nil {
		target = t
		targetResolved = true
	}

	// 如果 root 解析失败，使用原始路径检查
	if !rootResolved {
		// 检查原始路径是否在原始 root 内部
		origRoot := filepath.Clean(rootDir)
		origTarget := filepath.Clean(path)
		rootWithSep := origRoot + string(filepath.Separator)
		if origTarget != origRoot && !strings.HasPrefix(origTarget, rootWithSep) {
			return path
		}
		// 原始路径在 root 内部，继续处理
		rootDirClean = origRoot
		target = origTarget
	}

	// 如果 target 是符号链接但解析失败（可能指向不存在的路径），
	// 应该返回原路径，因为无法判断真实路径是否在 root 内部
	if !targetResolved {
		if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
			// 是符号链接但解析失败，返回原路径
			return path
		}
	}

	// 检查 target 是否在 root 内部
	rootWithSep := rootDirClean + string(filepath.Separator)
	if target != rootDirClean && !strings.HasPrefix(target, rootWithSep) {
		return path
	}

	// 计算相对路径
	rel, err := filepath.Rel(rootDirClean, target)
	if err != nil {
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
