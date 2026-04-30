package config

import (
	"os"
	"sync"

	"github.com/goccy/go-yaml"
)

// YamlProvider YAML 文件配置提供者
type YamlProvider struct {
	file string
	mu   sync.Mutex
}

// NewYamlProvider 创建 YAML 配置提供者
func NewYamlProvider(file string) *YamlProvider {
	return &YamlProvider{file: file}
}

func (y *YamlProvider) Type() string {
	return "yaml"
}

// Load 从 YAML 文件加载配置
func (y *YamlProvider) Load() (*Config, error) {
	data, err := os.ReadFile(y.file)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

// Save 将配置保存到 YAML 文件
func (y *YamlProvider) Save(conf *Config) error {
	y.mu.Lock()
	defer y.mu.Unlock()

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(y.file, data, 0644)
}
