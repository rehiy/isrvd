package config

import (
	"os"
	"strings"
)

// 全局配置实例
var global *Global

// 全局配置结构
type Global struct {
	Addr    string            // 监听地址
	BaseDir string            // 基础目录
	UserMap map[string]string // 用户名:明文密码
}

// 获取全局配置
func GetGlobal() *Global {
	if global == nil {
		initGlobal()
	}
	return global
}

// 初始化全局配置
func initGlobal() {
	cfg := &Global{
		Addr:    ":8080",
		BaseDir: ".",
		UserMap: make(map[string]string),
	}

	// 从环境变量读取端口
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		cfg.Addr = addr
	}

	// 从环境变量读取基础目录
	if baseDir := os.Getenv("BASE_DIR"); baseDir != "" {
		cfg.BaseDir = baseDir
	}

	// 从环境变量读取用户配置
	if usersEnv := os.Getenv("ADMIN_USERS"); usersEnv != "" {
		for _, pair := range strings.Split(usersEnv, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				cfg.UserMap[kv[0]] = kv[1]
			}
		}
	} else {
		cfg.UserMap["admin"] = "admin"
	}

	global = cfg
}
