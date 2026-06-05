package config

import (
	"os"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/logman"
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

	logman.Info("Loading YAML file", "file", y.file)

	conf := &Config{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	if migrate(conf, data) {
		if err := y.Save(conf); err != nil {
			logman.Warn("配置迁移保存失败", "error", err)
		} else {
			logman.Info("YAML 配置已自动更新（配置迁移）")
		}
	}

	return conf, nil
}

// Save 将配置保存到 YAML 文件
// 对 conf 做深拷贝，对副本还原相对路径后序列化，不影响原对象
func (y *YamlProvider) Save(conf *Config) error {
	y.mu.Lock()
	defer y.mu.Unlock()

	// 深拷贝：序列化再反序列化，得到独立副本
	buf, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	copy := &Config{}
	if err := yaml.Unmarshal(buf, copy); err != nil {
		return err
	}

	// 对副本做路径还原
	denormalizePaths(copy)

	data, err := yaml.Marshal(copy)
	if err != nil {
		return err
	}

	return os.WriteFile(y.file, data, 0644)
}
