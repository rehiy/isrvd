package cstore

import (
	"bytes"
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
	keyPath  string // etcd key，如 "/isrvd/config"
	fallback string // 可选：fallback YAML 文件路径，key 不存在时读取并写入 etcd
	timeout  time.Duration
}

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

// Get 读取 key 对应的值；etcd 中不存在时若配置了 fallback 文件则读取并回写。
func (e *EtcdStore) Get(key string) ([]byte, error) {
	return e.get(key)
}

func (e *EtcdStore) get(key string) ([]byte, error) {
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

	var val string
	var exists bool
	var err error
	// 保持锁定的旧版 libgo 可编译；空结果不得猜测为缺失并触发迁移。
	if lookup, ok := any(e.client).(interface {
		Lookup(context.Context, string) (string, bool, error)
	}); ok {
		val, exists, err = lookup.Lookup(ctx, e.etcdKey(key))
	} else {
		val, err = e.client.Get(ctx, e.etcdKey(key))
		exists = val != ""
		if err == nil && !exists {
			return nil, fmt.Errorf("cstore/etcd: 无法区分空值与缺失，请升级 libgo 以支持 Lookup")
		}
	}
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
	return e.put(key, value)
}

// CheckAndSet 使用 etcd 事务进行跨实例的值比较与条件写入。
func (e *EtcdStore) CheckAndSet(key string, expected, value []byte) error {
	client, ok := any(e.client).(interface {
		CompareAndPut(context.Context, string, []byte, []byte) (bool, error)
	})
	if !ok {
		return fmt.Errorf("cstore/etcd: 请升级 libgo 以支持 CompareAndPut，禁止无条件覆盖")
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	matched, err := client.CompareAndPut(ctx, e.etcdKey(key), expected, value)
	if err != nil {
		return err
	}
	if !matched {
		return ErrConflict
	}
	return nil
}

// put 执行无条件 etcd Put。
func (e *EtcdStore) put(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	if err := e.client.Put(ctx, e.etcdKey(key), string(value)); err != nil {
		return fmt.Errorf("cstore/etcd: Set %s 失败: %w", key, err)
	}
	return nil
}

// Watch 把事件作为变更提示，读取并发送最新状态；首次建立/重连时补偿连接间隙。
// 读取失败会重试，连续相同状态不重复发送；不重放所有历史事件。
func (e *EtcdStore) Watch(ctx context.Context, key string) <-chan Event {
	out := make(chan Event, 8)
	watchEvents, watchErrs := e.client.Watch(ctx, e.etcdKey(key))
	go func() {
		defer close(out)
		var last []byte
		known := false
		retry := time.NewTimer(time.Hour)
		retry.Stop()
		defer retry.Stop()
		var retryCh <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-watchEvents:
				if !ok {
					return
				}
			case <-retryCh:
			case err, ok := <-watchErrs:
				if !ok {
					watchErrs = nil
				} else {
					logman.Warn("cstore: Watch 连接错误", "key", key, "error", err)
				}
				continue
			}
			value, err := e.read(ctx, key)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logman.Warn("cstore: Watch 同步失败，稍后重试", "key", key, "error", err)
				retry.Reset(time.Second)
				retryCh = retry.C
				continue
			}
			retry.Stop()
			retryCh = nil
			if known && (value == nil) == (last == nil) && bytes.Equal(value, last) {
				continue
			}
			event := Event{Key: key, Type: EventPut, Value: value}
			if value == nil {
				event.Type = EventDelete
			}
			select {
			case out <- event:
				last, known = value, true
			case <-ctx.Done():
				return
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

func newEtcdStore(uri string) (*EtcdStore, error) {
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

	username := envOrDefault("ETCD_USERNAME", u.User.Username())
	password, _ := u.User.Password()
	password = envOrDefault("ETCD_PASSWORD", password)

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

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
