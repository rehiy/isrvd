package config

import (
	"os"
	"strings"
)

func init() {
	// 从环境变量读取端口
	if value := os.Getenv("LISTEN_ADDR"); value != "" {
		ListenAddr = value
	}

	// 从环境变量读取基础目录
	if value := os.Getenv("BASE_DIRECTORY"); value != "" {
		BaseDirectory = value
	}

	// 从环境变量读取用户配置
	if value := os.Getenv("MEMBERS"); value != "" {
		delete(Members, "admin") // 删除默认用户
		for _, pair := range strings.Split(value, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				Members[kv[0]] = kv[1]
			}
		}
	}
}
