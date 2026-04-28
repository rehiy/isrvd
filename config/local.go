package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// loadYAML 从指定路径加载完整 Config
func loadYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// saveYAML 将完整 Config 保存到指定路径
func saveYAML(path string, conf *Config) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// resolvePaths 处理 ContainerRoot 和 HomeDirectory 的相对路径
func resolvePaths(conf *Config) {
	if conf.Docker != nil && conf.Docker.ContainerRoot != "" && !filepath.IsAbs(conf.Docker.ContainerRoot) {
		conf.Docker.ContainerRoot = filepath.Join(conf.Server.RootDirectory, conf.Docker.ContainerRoot)
	}
	if conf.Members != nil {
		for _, m := range conf.Members {
			if m.HomeDirectory == "" {
				m.HomeDirectory = filepath.Join(conf.Server.RootDirectory, m.Username)
			} else if !filepath.IsAbs(m.HomeDirectory) {
				m.HomeDirectory = filepath.Join(conf.Server.RootDirectory, m.HomeDirectory)
			}
		}
	}
}
