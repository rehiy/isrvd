package config

import (
	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/secure"
	"github.com/rehiy/libgo/strutil"
)

// migrateFn 迁移函数类型：接收当前配置和原始 YAML 字节，返回是否发生了迁移
type migrateFn func(conf *Config, data []byte) bool

// migrations 按注册顺序依次执行的迁移函数列表，各函数之间相互独立、无依赖关系
var migrations = []migrateFn{
	migrateJWTSecret,
	migrateTHA,
	migratePasswords,
}

// migrate 依次执行所有迁移函数，返回是否有任何迁移发生
func migrate(conf *Config, data []byte) bool {
	migrated := false
	for _, fn := range migrations {
		if fn(conf, data) {
			migrated = true
		}
	}
	return migrated
}

// migrateJWTSecret 首次启动时自动生成 JWT 密钥
func migrateJWTSecret(conf *Config, _ []byte) bool {
	if conf.Server == nil || conf.Server.JWTSecret != "" {
		return false
	}
	conf.Server.JWTSecret = strutil.Rand(32)
	logman.Info("JWT 密钥已自动生成")
	return true
}

// migrateTHA 将旧版 server.proxyHeaderName/proxyTrustedCIDRs 迁移到 tha 配置段
func migrateTHA(conf *Config, data []byte) bool {
	// 已有 THA 配置段（无论是否启用），跳过迁移避免覆盖用户配置
	if conf.THA != nil {
		return false
	}

	// 用兼容结构解析旧字段
	legacy := &struct {
		Server *struct {
			ProxyHeaderName   string   `yaml:"proxyHeaderName"`
			ProxyTrustedCIDRs []string `yaml:"proxyTrustedCIDRs"`
		} `yaml:"server"`
	}{}

	if err := yaml.Unmarshal(data, legacy); err != nil || legacy.Server == nil {
		return false
	}
	if legacy.Server.ProxyHeaderName == "" {
		return false
	}

	logman.Info("检测到旧版 proxy 认证配置，自动迁移至 tha 配置段",
		"headerName", legacy.Server.ProxyHeaderName,
		"trustedCIDRs", legacy.Server.ProxyTrustedCIDRs,
	)

	conf.THA = &THAConfig{
		Enabled:      true,
		HeaderName:   legacy.Server.ProxyHeaderName,
		TrustedCIDRs: legacy.Server.ProxyTrustedCIDRs,
	}

	return true
}

// migratePasswords 迁移历史明文密码为加密格式
func migratePasswords(conf *Config, _ []byte) bool {
	migrated := false

	for _, m := range conf.Members {
		if m == nil || m.Password == "" || secure.IsBcrypt(m.Password) {
			continue
		}

		hashedPassword, err := secure.BcryptHash(m.Password)
		if err != nil {
			logman.Warn("密码加密失败", "username", m.Username, "error", err)
			continue
		}

		logman.Info("密码已自动迁移为加密格式", "username", m.Username)
		m.Password = hashedPassword
		migrated = true
	}

	return migrated
}
