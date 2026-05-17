package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rehiy/libgo/logman"
)

// ReloadCh 配置变更通知通道，etcd 变更时触发服务重载
var ReloadCh = make(chan struct{}, 1)

// ConfigProvider 配置提供者接口
type ConfigProvider interface {
	Type() string
	Load() (*Config, error)
	Save(*Config) error
}

type ConfigWatcher interface {
	Watch(context.Context) (<-chan struct{}, <-chan error)
}

var provider ConfigProvider

func Init() error {
	path := envOrDefault("CONFIG_PATH", "config.yml")
	switch {
	case strings.HasPrefix(strings.ToLower(path), "etcd://"):
		p, err := NewEtcdProvider(path)
		if err != nil {
			return err
		}
		provider = p
	case strings.HasPrefix(strings.ToLower(path), "file://"):
		provider = NewYamlProvider(strings.TrimPrefix(path, "file://"))
	case strings.Contains(path, "://"):
		return fmt.Errorf("不支持的配置路径: %s", path)
	default:
		provider = NewYamlProvider(path)
	}

	logman.Info("load config", "provider", provider.Type())
	if err := Load(); err != nil {
		return err
	}

	watchConfigChanges()
	return nil
}

// watchConfigChanges 监听 etcd 配置变更，变更时通知 server 层触发重载
func watchConfigChanges() {
	watcher, ok := provider.(ConfigWatcher)
	if !ok {
		return
	}

	changes, errs := watcher.Watch(context.Background())
	go func() {
		for changes != nil || errs != nil {
			select {
			case _, ok := <-changes:
				if !ok {
					changes = nil
					continue
				}
				logman.Info("Config changed, triggering reload", "provider", provider.Type())
				select {
				case ReloadCh <- struct{}{}:
				default:
				}
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				logman.Warn("Config watch error", "provider", provider.Type(), "error", err)
			}
		}
	}()
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
