package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/pango/logman"

	"isrvd/internal/helper"
)

var (
	// 模式
	Debug = false
	// 监听地址
	ListenAddr = ":8080"
	// JWT 密鑰
	JWTSecret = "default-secret-key"
	// 内网代理用户名 Header 名（为空则不启用）
	ProxyHeaderName = ""
	// 基础目录
	RootDirectory = "."
	// Agent LLM 配置
	Agent = &AgentConfig{}
	// Apisix 配置
	Apisix = &ApisixConfig{}
	// Docker 配置
	Docker = &DockerConfig{}
	// 应用市场配置
	Marketplace = &MarketplaceConfig{}
	// 工具栏链接配置
	Links []*LinkConfig
	// 成员配置
	Members = map[string]*MemberConfig{}
	// 当前加载的配置文件路径
	ConfigPath = ""
	// 版本信息，编译时通过脚本注入
	Version = "v0.0.0"
)

// 加载配置文件
func Load() error {
	file := os.Getenv("CONFIG_PATH")
	if file == "" {
		file = "config.yml"
	}
	ConfigPath = file

	// 读取配置文件
	logman.Info("load config", "file", file)
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
	ListenAddr = conf.Server.ListenAddr
	JWTSecret = conf.Server.JWTSecret
	ProxyHeaderName = conf.Server.ProxyHeaderName
	RootDirectory = conf.Server.RootDirectory

	// 更新 Agent 配置
	if conf.Agent != nil {
		Agent = conf.Agent
	}

	// 更新 Apisix 配置
	if conf.Apisix != nil {
		Apisix = conf.Apisix
	}

	// 更新 Docker 配置
	if conf.Docker != nil {
		Docker = conf.Docker
	}
	if !filepath.IsAbs(Docker.ContainerRoot) {
		Docker.ContainerRoot = filepath.Join(RootDirectory, Docker.ContainerRoot)
	}

	// 更新应用市场配置
	if conf.Marketplace != nil {
		Marketplace = conf.Marketplace
	}

	// 更新工具栏链接配置
	if conf.Links != nil {
		Links = conf.Links
	}

	// 更新 Member 配置
	Members = make(map[string]*MemberConfig, len(conf.Members))
	for _, m := range conf.Members {
		if m.HomeDirectory == "" {
			m.HomeDirectory = filepath.Join(RootDirectory, m.Username)
		} else if !filepath.IsAbs(m.HomeDirectory) {
			m.HomeDirectory = filepath.Join(RootDirectory, m.HomeDirectory)
		}
		Members[m.Username] = m
	}

	// 自动迁移明文密码为加密格式
	if err := migratePlaintextPasswords(); err != nil {
		logman.Warn("密码迁移失败", "error", err)
	}

	return nil
}

// migratePlaintextPasswords 自动迁移明文密码为加密格式
// 遍历所有成员，将明文密码替换为 bcrypt 格式的哈希值
// 迁移完成后自动保存配置文件
func migratePlaintextPasswords() error {
	needSave := false

	for _, m := range Members {
		// 跳过空密码或已经是加密格式的密码
		if m.Password == "" || helper.HashedBcrypt(m.Password) {
			continue
		}

		// 对明文密码进行加密
		hashedPassword, err := helper.HashPassword(m.Password)
		if err != nil {
			logman.Warn("密码加密失败", "username", m.Username, "error", err)
			continue
		}

		logman.Info("密码已自动迁移为加密格式", "username", m.Username)
		m.Password = hashedPassword
		needSave = true
	}

	// 如果有密码被迁移，自动保存配置文件
	if needSave {
		if err := Save(); err != nil {
			return err
		}
		logman.Info("配置文件已自动更新（密码迁移）")
	}

	return nil
}
