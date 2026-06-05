package cstore

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/etcd"
)

// EtcdStore 基于 etcd 的配置存储。
type EtcdStore struct {
	client    *etcd.Client
	keyPrefix string // etcd key 前缀，如 "/isrvd/"
	fallback  string // 可选：文件回退目录，key 不存在时从此目录读取并写入 etcd
	timeout   time.Duration
	mu        sync.Mutex
}

func newEtcdStore(uri string) (*EtcdStore, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("cstore/etcd: URI 解析失败: %w", err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("cstore/etcd: URI 缺少 endpoints")
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
			return nil, fmt.Errorf("cstore/etcd: timeout 无效: %w", err)
		}
	}

	username := envOrDefault("ETCD_USERNAME", u.User.Username())
	password, _ := u.User.Password()
	password = envOrDefault("ETCD_PASSWORD", password)

	// 确保 keyPrefix 以 "/" 结尾
	keyPrefix := u.Path
	if keyPrefix == "" || keyPrefix == "/" {
		keyPrefix = "/isrvd/"
	} else if !strings.HasSuffix(keyPrefix, "/") {
		keyPrefix += "/"
	}

	cli := etcd.New(etcd.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: timeout,
	})

	return &EtcdStore{
		client:    cli,
		keyPrefix: keyPrefix,
		fallback:  q.Get("fallback"),
		timeout:   timeout,
	}, nil
}

func (e *EtcdStore) etcdKey(key string) string {
	return e.keyPrefix + key
}

// Get 读取 key 对应的值；etcd 中不存在时若配置了 fallback 目录则从文件读取并回写。
func (e *EtcdStore) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	val, err := e.client.Get(ctx, e.etcdKey(key))
	if err != nil {
		return nil, fmt.Errorf("cstore/etcd: Get %s 失败: %w", key, err)
	}
	if val != "" {
		return []byte(val), nil
	}

	return e.getFromFallback(key)
}

// getFromFallback 从 fallback 目录读取文件并回写到 etcd。
func (e *EtcdStore) getFromFallback(key string) ([]byte, error) {
	if e.fallback == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(e.fallback, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cstore/etcd: 读取 fallback %s 失败: %w", key, err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.putLocked(key, data); err != nil {
		return nil, fmt.Errorf("cstore/etcd: 回写 fallback 到 etcd 失败: %w", err)
	}
	return data, nil
}

// Set 写入 key-value。
func (e *EtcdStore) Set(key string, value []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.putLocked(key, value)
}

// putLocked 执行 etcd Put，调用方须持有 e.mu。
func (e *EtcdStore) putLocked(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	if err := e.client.Put(ctx, e.etcdKey(key), string(value)); err != nil {
		return fmt.Errorf("cstore/etcd: Set %s 失败: %w", key, err)
	}
	return nil
}

// Watch 监听 key 变更，ctx 取消时 channel 关闭。
func (e *EtcdStore) Watch(ctx context.Context, key string) <-chan Event {
	out := make(chan Event, 8)
	watchEvents, watchErrs := e.client.Watch(ctx, e.etcdKey(key))

	go func() {
		defer close(out)
		for {
			select {
			case ev, ok := <-watchEvents:
				if !ok {
					return
				}
				event := Event{Key: key, Type: EventType(ev.Type)}
				if ev.Type == "PUT" {
					event.Value = []byte(ev.Value)
				}
				select {
				case out <- event:
				default:
				}
			case <-watchErrs:
				// 错误由 libgo/etcd 内部重连处理，忽略
			}
		}
	}()

	return out
}

// Close 释放资源。
func (e *EtcdStore) Close() error {
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
