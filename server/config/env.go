package config

import (
	"os"
	"strings"
)

func init() {
	// 从环境变量读取端口
	if value := os.Getenv("LISTEN_ADDR"); value != "" {
		Addr = value
	}

	// 从环境变量读取基础目录
	if value := os.Getenv("BASE_DIR"); value != "" {
		BaseDir = value
	}

	// 从环境变量读取用户配置
	if value := os.Getenv("ADMIN_USERS"); value != "" {
		for _, pair := range strings.Split(value, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				UserMap[kv[0]] = kv[1]
			}
		}
	} else {
		UserMap["admin"] = "admin"
	}
}
