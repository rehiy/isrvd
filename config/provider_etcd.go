package config

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/rehiy/libgo/etcd"
)

// EtcdProvider etcd 配置提供者
// CONFIG_PATH 示例：etcd://user:pass@host1:2379,host2:2379/isrvd/config?scheme=http&timeout=5s
type EtcdProvider struct {
	client   *etcd.Client
	key      string
	fallback string
	timeout  time.Duration
	mu       sync.Mutex
}

func NewEtcdProvider(path string) (*EtcdProvider, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, fmt.Errorf("etcd 配置路径缺少 endpoints")
	}

	q := u.Query()
	scheme := q.Get("scheme")
	if scheme == "" {
		scheme = "http"
	}

	var endpoints []string
	for _, host := range strings.Split(u.Host, ",") {
		if host = strings.TrimSpace(host); host != "" {
			endpoints = append(endpoints, scheme+"://"+host)
		}
	}

	timeout := 5 * time.Second
	if raw := q.Get("timeout"); raw != "" {
		if timeout, err = time.ParseDuration(raw); err != nil {
			return nil, fmt.Errorf("etcd timeout 无效: %w", err)
		}
	}

	username := envOrDefault("ETCD_USERNAME", u.User.Username())
	password, _ := u.User.Password()
	password = envOrDefault("ETCD_PASSWORD", password)

	key := u.Path
	if key == "" || key == "/" {
		key = "/isrvd/config"
	}

	cli := etcd.New(etcd.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: timeout,
	})

	return &EtcdProvider{
		client:   cli,
		key:      key,
		fallback: q.Get("fallback"),
		timeout:  timeout,
	}, nil
}

func (e *EtcdProvider) Type() string {
	return "etcd"
}

func (e *EtcdProvider) Load() (*Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	val, err := e.client.Get(ctx, e.key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return e.loadFallback()
	}

	conf := &Config{}
	if err := yaml.Unmarshal([]byte(val), conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func (e *EtcdProvider) loadFallback() (*Config, error) {
	if e.fallback == "" {
		return nil, fmt.Errorf("etcd 配置不存在: %s", e.key)
	}
	conf, err := NewYamlProvider(e.fallback).Load()
	if err != nil {
		return nil, fmt.Errorf("读取 fallback 配置失败: %w", err)
	}
	if err := e.Save(conf); err != nil {
		return nil, fmt.Errorf("写入 etcd fallback 配置失败: %w", err)
	}
	return conf, nil
}

func (e *EtcdProvider) Watch(ctx context.Context) (<-chan struct{}, <-chan error) {
	changes := make(chan struct{}, 1)
	errs := make(chan error, 1)

	watchEvents, watchErrs := e.client.Watch(ctx, e.key)

	go func() {
		defer close(changes)
		defer close(errs)

		for {
			select {
			case ev, ok := <-watchEvents:
				if !ok {
					return
				}
				switch ev.Type {
				case "PUT":
					var conf Config
					if err := yaml.Unmarshal([]byte(ev.Value), &conf); err != nil {
						select {
						case errs <- fmt.Errorf("etcd 配置解析失败: %w", err):
						default:
						}
						continue
					}
					select {
					case changes <- struct{}{}:
					default:
					}
				case "DELETE":
					select {
					case errs <- fmt.Errorf("etcd 配置已删除: %s", e.key):
					default:
					}
				}
			case err, ok := <-watchErrs:
				if !ok {
					return
				}
				select {
				case errs <- err:
				default:
				}
			}
		}
	}()

	return changes, errs
}

func (e *EtcdProvider) Save(conf *Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	return e.client.Put(ctx, e.key, string(data))
}
