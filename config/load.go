package config

import (
	"context"
	"encoding/json"
	"os"
	"time"

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
	// etcd 连接配置（从 YAML 读取保留）
	Etcd *EtcdConfig
	// 版本信息，编译时通过脚本注入
	Version = "v0.0.0"
)

var remoteStore RemoteStore

// 加载配置文件
func Load() error {
	file := os.Getenv("CONFIG_PATH")
	if file == "" {
		file = "config.yml"
	}
	ConfigPath = file

	// 1. 读取本地 YAML
	conf, err := loadYAML(file)
	if err != nil {
		return err
	}

	// 填充默认值
	if conf.Server == nil {
		conf.Server = &Server{}
	}
	if conf.Agent == nil {
		conf.Agent = &AgentConfig{}
	}
	if conf.Apisix == nil {
		conf.Apisix = &ApisixConfig{}
	}
	if conf.Docker == nil {
		conf.Docker = &DockerConfig{}
	}
	if conf.Marketplace == nil {
		conf.Marketplace = &MarketplaceConfig{}
	}

	// 2. 如果配置了 etcd，初始化远程存储并预加载
	if conf.Etcd != nil && len(conf.Etcd.Endpoints) > 0 {
		store, err := newEtcdStore(conf.Etcd)
		if err != nil {
			logman.Warn("etcd 初始化失败，使用本地配置", "error", err)
		} else {
			remoteStore = store
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			rc, rev, err := store.Load(ctx)
			cancel()
			if err != nil {
				logman.Warn("etcd 预加载失败，使用本地配置", "error", err)
			} else {
				if err := validateRemote(rc); err != nil {
					logman.Warn("etcd 配置校验失败，使用本地配置", "error", err)
				} else {
					mergeRemote(conf, rc)
					logman.Info("etcd 配置已加载", "revision", rev)
					go startWatch(rev)
				}
			}
		}
	}

	// 3. 处理路径
	resolvePaths(conf)

	// 4. 更新全局变量
	Debug = conf.Server.Debug
	ListenAddr = conf.Server.ListenAddr
	JWTSecret = conf.Server.JWTSecret
	ProxyHeaderName = conf.Server.ProxyHeaderName
	RootDirectory = conf.Server.RootDirectory

	if conf.Agent != nil {
		Agent = conf.Agent
	}
	if conf.Apisix != nil {
		Apisix = conf.Apisix
	}
	if conf.Docker != nil {
		Docker = conf.Docker
	}
	if conf.Marketplace != nil {
		Marketplace = conf.Marketplace
	}
	if conf.Links != nil {
		Links = conf.Links
	}

	Members = make(map[string]*MemberConfig, len(conf.Members))
	for _, m := range conf.Members {
		Members[m.Username] = m
	}

	Etcd = conf.Etcd

	// 5. 自动迁移明文密码
	if err := migratePlaintextPasswords(); err != nil {
		logman.Warn("密码迁移失败", "error", err)
	}

	return nil
}

func startWatch(rev int64) {
	if remoteStore == nil {
		return
	}
	ctx := context.Background()
	err := remoteStore.Watch(ctx, rev, func(key string, value []byte) {
		// DELETE event: reset to empty
		if value == nil && key != "_compacted" && key != "_canceled" {
			switch key {
			case "agent":
				Agent = &AgentConfig{}
			case "apisix":
				Apisix.AdminKey = ""
			case "marketplace":
				Marketplace = &MarketplaceConfig{}
			case "links":
				Links = []*LinkConfig{}
			case "members":
				Members = map[string]*MemberConfig{}
			case "docker":
				Docker.Registries = []*DockerRegistry{}
			case "server":
				JWTSecret = ""
				ProxyHeaderName = ""
			}
			logman.Info("etcd 配置已删除", "key", key)
			return
		}
		switch key {
		case "_compacted", "_canceled":
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			rc, _, err := remoteStore.Load(ctx2)
			cancel()
			if err != nil {
				logman.Error("etcd 全量重载失败", "error", err)
				return
			}
			applyRemoteToGlobals(rc)
			logman.Info("etcd 配置已全量重载", "trigger", key)
		case "agent":
			var a AgentConfig
			if err := json.Unmarshal(value, &a); err != nil {
				logman.Error("etcd watch unmarshal agent failed", "error", err)
				return
			}
			Agent = &a
			logman.Info("etcd 配置已热更新", "key", key)
		case "apisix":
			var a ApisixRemote
			if err := json.Unmarshal(value, &a); err != nil {
				logman.Error("etcd watch unmarshal apisix failed", "error", err)
				return
			}
			Apisix.AdminKey = a.AdminKey
			logman.Info("etcd 配置已热更新", "key", key)
		case "marketplace":
			var m MarketplaceConfig
			if err := json.Unmarshal(value, &m); err != nil {
				logman.Error("etcd watch unmarshal marketplace failed", "error", err)
				return
			}
			Marketplace = &m
			logman.Info("etcd 配置已热更新", "key", key)
		case "links":
			var l []*LinkConfig
			if err := json.Unmarshal(value, &l); err != nil {
				logman.Error("etcd watch unmarshal links failed", "error", err)
				return
			}
			Links = l
			logman.Info("etcd 配置已热更新", "key", key)
		case "members":
			var ms []*MemberConfig
			if err := json.Unmarshal(value, &ms); err != nil {
				logman.Error("etcd watch unmarshal members failed", "error", err)
				return
			}
			Members = make(map[string]*MemberConfig, len(ms))
			for _, m := range ms {
				Members[m.Username] = m
			}
			logman.Info("etcd 配置已热更新", "key", key)
		case "docker":
			var d DockerRemote
			if err := json.Unmarshal(value, &d); err != nil {
				logman.Error("etcd watch unmarshal docker failed", "error", err)
				return
			}
			Docker.Registries = d.Registries
			logman.Info("etcd 配置已热更新", "key", key)
		case "server":
			var s ServerRemote
			if err := json.Unmarshal(value, &s); err != nil {
				logman.Error("etcd watch unmarshal server failed", "error", err)
				return
			}
			JWTSecret = s.JWTSecret
			ProxyHeaderName = s.ProxyHeaderName
			logman.Info("etcd 配置已热更新", "key", key)
		}
	})
	if err != nil {
		logman.Error("etcd watch 异常退出", "error", err)
	}
}

func applyRemoteToGlobals(rc *RemoteConfig) {
	if rc.Agent != nil {
		Agent = rc.Agent
	}
	if rc.Apisix != nil {
		Apisix.AdminKey = rc.Apisix.AdminKey
	}
	if rc.Marketplace != nil {
		Marketplace = rc.Marketplace
	}
	if rc.Links != nil {
		Links = rc.Links
	}
	if rc.Members != nil {
		Members = make(map[string]*MemberConfig, len(rc.Members))
		for _, m := range rc.Members {
			Members[m.Username] = m
		}
	}
	if rc.Docker != nil && rc.Docker.Registries != nil {
		Docker.Registries = rc.Docker.Registries
	}
	if rc.Server != nil {
		JWTSecret = rc.Server.JWTSecret
		ProxyHeaderName = rc.Server.ProxyHeaderName
	}
}

// migratePlaintextPasswords 自动迁移明文密码为加密格式
func migratePlaintextPasswords() error {
	needSave := false
	for _, m := range Members {
		if m.Password == "" || helper.HashedBcrypt(m.Password) {
			continue
		}
		hashedPassword, err := helper.HashPassword(m.Password)
		if err != nil {
			logman.Warn("密码加密失败", "username", m.Username, "error", err)
			continue
		}
		logman.Info("密码已自动迁移为加密格式", "username", m.Username)
		m.Password = hashedPassword
		needSave = true
	}
	if needSave {
		if err := Save(); err != nil {
			return err
		}
		logman.Info("配置文件已自动更新（密码迁移）")
	}
	return nil
}
