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
	// Docker 配置
	Docker = &DockerConfig{}
	// 成员配置
	Members = map[string]*MemberConfig{}
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

	// 更新 Docker 配置
	Docker = conf.Docker
	if value := os.Getenv("DOCKER_HOST"); value != "" {
		Docker.Host = value
	}
	if value := os.Getenv("DOCKER_CONTAINER_ROOT"); value != "" {
		Docker.ContainerRoot = value
	}
	if !filepath.IsAbs(Docker.ContainerRoot) {
		Docker.ContainerRoot = filepath.Join(RootDirectory, Docker.ContainerRoot)
	}

	// 更新成员配置
	for _, m := range conf.Members {
		if !filepath.IsAbs(m.HomeDirectory) {
			m.HomeDirectory = filepath.Join(RootDirectory, m.HomeDirectory)
		}
		if err := os.MkdirAll(m.HomeDirectory, 0755); err != nil {
			return err
		}
		Members[m.Username] = m
	}

	return nil
}
