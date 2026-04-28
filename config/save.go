package config

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var saveMu sync.Mutex

// Save 将当前全局配置回写到配置文件
func Save() error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if ConfigPath == "" {
		return fmt.Errorf("config path not initialized")
	}

	// 1. 如果 etcd 可用，先保存全局段到 etcd
	if remoteStore != nil {
		conf := buildConfigFromGlobals()
		rc := extractRemote(conf)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := remoteStore.Save(ctx, rc)
		cancel()
		if err != nil {
			return fmt.Errorf("保存 etcd 配置失败: %w", err)
		}
	}

	// 2. 保存完整配置到本地 YAML（作为 fallback）
	conf := buildConfigFromGlobals()
	if err := saveYAML(ConfigPath, conf); err != nil {
		return fmt.Errorf("保存本地配置失败: %w", err)
	}

	return nil
}

// buildConfigFromGlobals 从全局变量组装完整 Config
func buildConfigFromGlobals() *Config {
	members := make([]*MemberConfig, 0, len(Members))
	for _, m := range Members {
		members = append(members, m)
	}
	return &Config{
		Server: &Server{
			Debug:           Debug,
			ListenAddr:      ListenAddr,
			JWTSecret:       JWTSecret,
			ProxyHeaderName: ProxyHeaderName,
			RootDirectory:   RootDirectory,
		},
		Agent:       Agent,
		Apisix:      Apisix,
		Docker:      Docker,
		Marketplace: Marketplace,
		Links:       Links,
		Members:     members,
	}
}
