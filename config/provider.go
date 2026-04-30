package config

import (
	"os"

	"github.com/rehiy/pango/logman"
)

// 全局配置提供者实例
var provider ConfigProvider

// ConfigProvider 配置提供者接口
// 支持从不同数据源加载配置（yml/etcd 等）
type ConfigProvider interface {
	Type() string
	Load() (*Config, error)
	Save(*Config) error
}

func Init() error {
	if provider == nil {
		file := "config.yml"
		if f := os.Getenv("CONFIG_PATH"); f != "" {
			file = f
		}
		provider = NewYamlProvider(file)
	}

	logman.Info("load config", "provider", provider.Type())
	return Load()
}
