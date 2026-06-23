package config

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/cstore"
)

// ReloadCh 配置变更通知通道，etcd 变更时触发服务重载
var ReloadCh = make(chan struct{}, 1)

// store 全局配置存储实例
var store *cstore.TypedStore[*Config]

func Init() error {
	uri := envOrDefault("CONFIG_PATH", "config.yml")

	var err error
	store, err = cstore.NewTypedFromPath[*Config](uri)
	if err != nil {
		return err
	}

	if !strings.Contains(uri, "://") {
		if abs, err := filepath.Abs(uri); err == nil {
			uri = abs
		}
	}
	logman.Info("load config", "path", uri)
	if err := Load(); err != nil {
		return err
	}

	watchConfigChanges()
	return nil
}

// Load 从 store 加载配置并应用到全局变量
func Load() error {
	conf, err := store.Get()
	if err != nil {
		return err
	}
	if conf == nil {
		logman.Warn("未找到配置文件", "key", store.Key())
		Apply(nil)
		return nil
	}

	if raw, _ := store.Store().Get(store.Key()); migrate(conf, raw) {
		if err := store.Set(conf); err != nil {
			logman.Warn("配置迁移保存失败", "error", err)
		} else {
			logman.Info("配置已自动更新（配置迁移）")
		}
	}

	Apply(conf)
	return nil
}

// Save 将当前全局配置保存到 store
func Save() error {
	members := make([]*MemberConfig, 0, len(Members))
	for _, m := range Members {
		members = append(members, m)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Username < members[j].Username
	})

	conf := &Config{
		Schema:      schema,
		Server:      Server,
		Password:    Password,
		THA:         THA,
		OIDC:        OIDC,
		Passkey:     Passkey,
		Agent:       Agent,
		Apisix:      Apisix,
		Caddy:       Caddy,
		Docker:      Docker,
		Monitor:     Monitor,
		Marketplace: Marketplace,
		Links:       Links,
		Members:     members,
	}

	// 深拷贝后 denormalize，避免修改全局指针指向的对象
	buf, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	var snapshot Config
	if err := yaml.Unmarshal(buf, &snapshot); err != nil {
		return err
	}
	denormalizePaths(&snapshot)
	return store.Set(&snapshot)
}

// --- 辅助函数 ---

func watchConfigChanges() {
	ch := store.Watch(context.Background())
	if ch == nil {
		return
	}
	go func() {
		for ev := range ch {
			switch ev.Type {
			case cstore.EventPut:
				logman.Info("Config changed, triggering reload", "key", ev.Key)
				select {
				case ReloadCh <- struct{}{}:
				default:
				}
			case cstore.EventDelete:
				logman.Warn("Config deleted in etcd", "key", ev.Key)
			}
		}
	}()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
