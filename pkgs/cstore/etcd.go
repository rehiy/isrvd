package cstore

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rehiy/libgo/etcd"
	"github.com/rehiy/libgo/logman"
)

// EtcdStore 基于 etcd 的配置存储。
type EtcdStore struct {
	client   *etcd.Client
	keyPath  string // etcd key，如 "/app/config"
	fallback string // 可选：fallback YAML 文件路径，key 不存在时读取并写入 etcd
	timeout  time.Duration
}

func newEtcdStore(uri string, opt options) (*EtcdStore, error) {
	// url.Parse 不支持逗号分隔的多个 host:port，先取出 host 段，再用占位 host 解析其余部分
	rest := uri[len("etcd://"):]
	authority := rest[:strings.IndexAny(rest+"/", "/?")]
	at := strings.LastIndex(authority, "@") + 1
	userinfo, hosts := authority[:at], authority[at:]
	if hosts == "" {
		return nil, fmt.Errorf("cstore/etcd: URI 缺少 endpoints")
	}
	u, err := url.Parse("etcd://" + userinfo + "endpoints" + rest[len(authority):])
	if err != nil {
		return nil, fmt.Errorf("cstore/etcd: URI 解析失败: %w", err)
	}

	q := u.Query()
	scheme := q.Get("scheme")
	if scheme == "" {
		scheme = "http"
	}

	var endpoints []string
	for _, host := range strings.Split(hosts, ",") {
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

	username := u.User.Username()
	if opt.etcdUsername != "" {
		username = opt.etcdUsername
	}
	password, _ := u.User.Password()
	if opt.etcdPassword != "" {
		password = opt.etcdPassword
	}

	keyPath := u.Path
	if keyPath == "" || keyPath == "/" {
		return nil, fmt.Errorf("cstore/etcd: URI 缺少配置 key")
	}

	cli := etcd.New(etcd.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: timeout,
	})

	return &EtcdStore{
		client:   cli,
		keyPath:  keyPath,
		fallback: q.Get("fallback"),
		timeout:  timeout,
	}, nil
}

// Get 读取 key 对应的值；etcd 中不存在时若配置了 fallback 文件则读取并回写。
func (e *EtcdStore) Get(key string) ([]byte, error) {
	data, err := e.read(context.Background(), key)
	if err != nil || data != nil {
		return data, err
	}
	return e.getFromFallback(key)
}

// read 只读取后端；提交前检查不得触发 fallback 回写。
func (e *EtcdStore) read(parent context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, e.timeout)
	defer cancel()

	val, exists, err := e.client.Lookup(ctx, e.etcdKey(key))
	if err != nil {
		return nil, fmt.Errorf("cstore/etcd: Get %s 失败: %w", key, err)
	}
	if exists {
		return []byte(val), nil
	}

	return nil, nil
}

// getFromFallback 从 fallback 文件读取并回写到 etcd。
func (e *EtcdStore) getFromFallback(key string) ([]byte, error) {
	if e.fallback == "" {
		return nil, nil
	}
	data, err := os.ReadFile(e.fallbackPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cstore/etcd: 读取 fallback %s 失败: %w", key, err)
	}
	if err := e.CheckAndSet(key, nil, data); err != nil {
		if errors.Is(err, ErrConflict) {
			return e.read(context.Background(), key)
		}
		return nil, err
	}
	return data, nil
}

// Set 写入 key-value。
func (e *EtcdStore) Set(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	if err := e.client.Put(ctx, e.etcdKey(key), string(value)); err != nil {
		return fmt.Errorf("cstore/etcd: Set %s 失败: %w", key, err)
	}
	return nil
}

// CheckAndSet 使用 etcd 事务进行跨实例的值比较与条件写入。
func (e *EtcdStore) CheckAndSet(key string, expected, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	matched, err := e.client.CompareAndPut(ctx, e.etcdKey(key), expected, value)
	if err != nil {
		return err
	}
	if !matched {
		return ErrConflict
	}
	return nil
}

// Watch 将 etcd 最新状态转换为存储事件，ctx 取消时停止。
func (e *EtcdStore) Watch(ctx context.Context, key string) <-chan Event {
	out := make(chan Event, 8)
	events, errs := e.client.Watch(ctx, e.etcdKey(key))
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errs:
				if !ok {
					errs = nil
				} else {
					logman.Warn("cstore: Watch 错误", "key", key, "error", err)
				}
			case state, ok := <-events:
				if !ok {
					return
				}
				event := Event{Key: key, Type: EventDelete}
				if state.Type == "PUT" {
					event.Type, event.Value = EventPut, []byte(state.Value)
				}
				select {
				case out <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// Close 释放资源。
func (e *EtcdStore) Close() error {
	return nil
}

// ─── 辅助函数 ───

func (e *EtcdStore) etcdKey(key string) string {
	if key == "" {
		return e.keyPath
	}
	return strings.TrimSuffix(e.keyPath, "/") + "/" + key
}

func (e *EtcdStore) fallbackPath(key string) string {
	if key == "" || filepath.Ext(e.fallback) != "" {
		return e.fallback
	}
	return filepath.Join(e.fallback, key)
}
