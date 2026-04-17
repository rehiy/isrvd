package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/goccy/go-yaml"
)

var saveMu sync.Mutex

// Save 将当前全局配置回写到配置文件
func Save() error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if ConfigPath == "" {
		return fmt.Errorf("config path not initialized")
	}

	members := make([]*MemberConfig, 0, len(Members))
	for _, m := range Members {
		members = append(members, m)
	}

	conf := &Config{
		Server: &Server{
			Debug:           Debug,
			ListenAddr:      ListenAddr,
			JWTSecret:       JWTSecret,
			ProxyHeaderName: ProxyHeaderName,
			RootDirectory:   RootDirectory,
		},
		Agent:   Agent,
		Apisix:  Apisix,
		Docker:  Docker,
		Members: members,
	}

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath, data, 0644)
}
