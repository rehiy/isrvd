package config

import (
	"os"
	"strings"
)

// Config 应用配置结构
type Config struct {
	BaseDir string            // 基础目录
	UserMap map[string]string // 用户名:明文密码
	Port    string            // 服务端口
}

// Global 全局配置实例
var Global *Config

// Init 初始化配置
func Init() *Config {
	cfg := &Config{
		BaseDir: ".",
		UserMap: make(map[string]string),
		Port:    "8080",
	}

	// 从环境变量读取基础目录
	if baseDir := os.Getenv("BASE_DIR"); baseDir != "" {
		cfg.BaseDir = baseDir
	}

	// 从环境变量读取端口
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	// 从环境变量读取用户配置
	if usersEnv := os.Getenv("USERS"); usersEnv != "" {
		for _, pair := range strings.Split(usersEnv, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				cfg.UserMap[kv[0]] = kv[1]
			}
		}
	} else {
		// 默认用户
		cfg.UserMap["admin"] = "admin"
	}

	Global = cfg
	return cfg
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if Global == nil {
		return Init()
	}
	return Global
}
