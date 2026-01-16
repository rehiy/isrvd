package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 模式
var Debug = false

// 监听地址
var ListenAddr = ":8080"

// 基础目录
var RootDirectory = "."

// 成员配置
var Members = make(map[string]*Member)

// 加载配置文件
func Load() error {
	file := os.Getenv("CONFIG_PATH")
	if file == "" {
		file = "config.yml"
	}

	// 读取配置文件
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// 解析配置文件
	conf := &Config{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return err
	}

	// 更新全局变量
	Debug = conf.Server.Debug
	if value := os.Getenv("DEBUG"); value != "" {
		Debug = value == "true"
	}
	ListenAddr = conf.Server.ListenAddr
	if value := os.Getenv("LISTEN_ADDR"); value != "" {
		ListenAddr = value
	}
	RootDirectory = conf.Server.RootDirectory
	if value := os.Getenv("ROOT_DIRECTORY"); value != "" {
		RootDirectory = value
	}

	// 更新成员配置
	for _, m := range conf.Members {
		m.HomeDirectory = filepath.Join(RootDirectory, m.HomeDirectory)
		Members[m.Username] = m
	}

	return nil
}
