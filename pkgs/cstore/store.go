// Package cstore 提供统一的配置存储抽象，支持本地文件和 etcd 两种后端。
//
// key 语义：配置文件名（如 "config.yml"），FileStore 拼接到 base 目录，EtcdStore 拼接到 key 前缀。
//
// Open 传入目录/前缀 URI：
//   - etcd://[user:pass@]host:port[,host:port][/prefix][?scheme=http&timeout=5s&fallback=/path/to/dir]
//   - file:///abs/dir 或 /abs/dir 或 rel/dir  → FileStore
//
// OpenWithKey 传入完整文件路径，自动拆分目录和文件名：
//   - /abs/path/to/config.yml 或 rel/config.yml  → FileStore
//   - file:///abs/path/to/config.yml             → FileStore
//   - etcd://...                                 → EtcdStore，key = defaultKey
package cstore

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// EventType 变更事件类型
type EventType string

const (
	EventPut    EventType = "PUT"
	EventDelete EventType = "DELETE"
)

// Event 配置变更事件
type Event struct {
	Key   string    // 变更的 key（文件名）
	Value []byte    // PUT 时为新值，DELETE 时为 nil
	Type  EventType // PUT | DELETE
}

// Store 配置存储接口
type Store interface {
	// Get 读取 key 对应的值，key 不存在时返回 nil, nil
	Get(key string) ([]byte, error)
	// Set 写入 key-value
	Set(key string, value []byte) error
	// Watch 监听 key 变更，ctx 取消时 channel 关闭；FileStore 返回 nil channel
	Watch(ctx context.Context, key string) <-chan Event
	// Close 释放资源
	Close() error
}

// Open 根据 URI 创建 Store，URI 指向目录（或 etcd 前缀）。
//
//   - etcd://...  → EtcdStore
//   - file://...  → FileStore（取 path 部分作为 base 目录）
//   - 其他        → FileStore（URI 直接作为 base 目录路径）
func Open(uri string) (Store, error) {
	lower := strings.ToLower(uri)
	switch {
	case strings.HasPrefix(lower, "etcd://"):
		return newEtcdStore(uri)
	case strings.HasPrefix(lower, "file://"):
		return newFileStore(strings.TrimPrefix(uri, "file://"))
	case strings.Contains(uri, "://"):
		return nil, fmt.Errorf("cstore: 不支持的存储 URI: %s", uri)
	default:
		return newFileStore(uri)
	}
}

// OpenWithKey 根据完整文件路径或 etcd URI 创建 Store，并返回对应的 key。
// 调用方无需自行解析 URI，直接将 CONFIG_PATH 等环境变量传入即可。
//
// 解析规则：
//   - etcd://...                → EtcdStore，key = defaultKey
//   - file:///path/to/file.yml  → FileStore(dir)，key = file.yml
//   - /path/to/file.yml         → FileStore(dir)，key = file.yml
//   - rel/file.yml              → FileStore(dir)，key = file.yml
func OpenWithKey(uri, defaultKey string) (Store, string, error) {
	lower := strings.ToLower(uri)
	switch {
	case strings.HasPrefix(lower, "etcd://"):
		s, err := newEtcdStore(uri)
		return s, defaultKey, err
	case strings.HasPrefix(lower, "file://"):
		filePath := strings.TrimPrefix(uri, "file://")
		s, err := newFileStore(filepath.Dir(filePath))
		return s, filepath.Base(filePath), err
	case strings.Contains(uri, "://"):
		return nil, "", fmt.Errorf("cstore: 不支持的存储 URI: %s", uri)
	default:
		s, err := newFileStore(filepath.Dir(uri))
		return s, filepath.Base(uri), err
	}
}
