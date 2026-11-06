package openapi

// ProjectConfig 定义项目的目录结构和 OpenAPI 元数据
// 修改 DefaultConfig() 即可适配不同项目
type ProjectConfig struct {
	// 目录配置（相对于项目根目录）
	CtrlDir        string   // handler 文件目录
	CtrlPrefix     string   // handler 文件名前缀
	ServiceDir     string   // service 层目录
	TypeDirs       []string // 额外的类型定义目录
	ResponseStruct string   // 响应外壳 schema 名称

	// OpenAPI 元数据
	Title      string // API 标题
	Version    string // API 版本
	ServerURL  string // 服务 URL 前缀
	ServerDesc string // 服务描述

	// 输出
	OutputFile string // 输出文件路径
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ProjectConfig {
	return &ProjectConfig{
		CtrlDir:        "server/app",
		CtrlPrefix:     "ctrl_",
		ServiceDir:     "server/service",
		TypeDirs:       []string{"server/config", "pkgs"},
		ResponseStruct: "APIResponse",
		Title:          "isrvd API",
		Version:        "1.0.0",
		ServerURL:      "/api",
		ServerDesc:     "API 服务",
		OutputFile:     "public/openapi/data.json",
	}
}
