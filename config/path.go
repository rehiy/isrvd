package config

import "path/filepath"

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

// PathToRel 将绝对路径还原为基于 rootDir 的相对路径（"./" 前缀）。
// 仅当 path 在 rootDir 内部时才转换；rootDir 外的路径原样返回。
func PathToRel(rootDir, path string) string {
	if path == "" || !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(filepath.Clean(rootDir), filepath.Clean(path))
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return path
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
