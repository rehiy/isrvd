package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

var (
	// 模式
	Debug = false
	// 监听地址
	ListenAddr = ":8080"
	// JWT 密钥
	JWTSecret = "default-secret-key"
	// 内网代理用户名 Header 名（为空则不启用）
	ProxyHeaderName = ""
	// 基础目录
	RootDirectory = "."
	// 容器数据根目录
	ContainerRoot = ""
	// 成员配置
	Members = map[string]*Member{}
)

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
	JWTSecret = conf.Server.JWTSecret
	if value := os.Getenv("JWT_SECRET"); value != "" {
		JWTSecret = value
	}
	ProxyHeaderName = conf.Server.ProxyHeaderName
	if value := os.Getenv("PROXY_HEADER_NAME"); value != "" {
		ProxyHeaderName = value
	}
	RootDirectory = conf.Server.RootDirectory
	if value := os.Getenv("ROOT_DIRECTORY"); value != "" {
		RootDirectory = value
	}
	ContainerRoot = conf.Server.ContainerRoot
	if value := os.Getenv("CONTAINER_ROOT"); value != "" {
		ContainerRoot = value
	}
	if !filepath.IsAbs(ContainerRoot) {
		ContainerRoot = filepath.Join(RootDirectory, ContainerRoot)
	}

	// 更新成员配置
	for _, m := range conf.Members {
		if !filepath.IsAbs(m.HomeDirectory) {
			m.HomeDirectory = filepath.Join(RootDirectory, m.HomeDirectory)
		}
		Members[m.Username] = m
	}

	// 自动创建用户目录
	for _, m := range Members {
		if err := os.MkdirAll(m.HomeDirectory, 0755); err != nil {
			return err
		}
	}

	return nil
}
