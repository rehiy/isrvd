package models

// Global 全局配置结构
type Global struct {
	Addr    string            // 监听地址
	BaseDir string            // 基础目录
	UserMap map[string]string // 用户名:明文密码
}
