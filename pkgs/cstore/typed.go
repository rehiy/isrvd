package cstore

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/logman"
)

// TypedStore 在 Store 之上封装了 YAML 序列化/反序列化，
// 业务层直接操作类型化的值，不再感知 []byte 和 yaml 细节。
type TypedStore[T any] struct {
	store Store
	key   string
}

// NewTyped 创建 TypedStore。uri 指向目录或 etcd 前缀，key 为文件名（如 "config.yml"）。
func NewTyped[T any](uri, key string) (*TypedStore[T], error) {
	s, err := Open(uri)
	if err != nil {
		return nil, err
	}
	return &TypedStore[T]{store: s, key: key}, nil
}

// NewTypedFromPath 根据完整文件路径或 etcd URI 创建 TypedStore，自动拆分目录和文件名。
// etcd URI 使用 URI path 作为完整 key（必填）；文件路径取 Base 作为 key。
func NewTypedFromPath[T any](path string) (*TypedStore[T], error) {
	s, key, err := OpenWithKey(path)
	if err != nil {
		return nil, err
	}
	return &TypedStore[T]{store: s, key: key}, nil
}

// WrapTyped 用已有的 Store 创建 TypedStore，适合多个 key 共用同一 Store 实例。
func WrapTyped[T any](s Store, key string) *TypedStore[T] {
	return &TypedStore[T]{store: s, key: key}
}

// Get 读取并反序列化值。key 不存在时返回零值和 nil error。
func (t *TypedStore[T]) Get() (val T, err error) {
	data, err := t.store.Get(t.key)
	if err != nil {
		return val, fmt.Errorf("cstore: Get %s 失败: %w", t.key, err)
	}
	if data == nil {
		return val, nil
	}
	if err := unmarshalVal(data, &val); err != nil {
		return val, fmt.Errorf("cstore: 解析 %s 失败: %w", t.key, err)
	}
	return val, nil
}

// Set 序列化并写入值。
func (t *TypedStore[T]) Set(val T) error {
	data, err := yaml.Marshal(val)
	if err != nil {
		return fmt.Errorf("cstore: 序列化 %s 失败: %w", t.key, err)
	}
	return t.store.Set(t.key, data)
}

// TypedEvent 类型化的变更事件。
type TypedEvent[T any] struct {
	Key   string
	Type  EventType
	Value T    // PUT 时有效
	Valid bool // Value 是否成功反序列化
}

// Watch 监听 key 变更，返回类型化事件 channel。FileStore 返回 nil。
func (t *TypedStore[T]) Watch(ctx context.Context) <-chan TypedEvent[T] {
	raw := t.store.Watch(ctx, t.key)
	if raw == nil {
		return nil
	}

	out := make(chan TypedEvent[T], 8)
	go func() {
		defer close(out)
		for ev := range raw {
			te := TypedEvent[T]{Type: ev.Type, Key: ev.Key}
			if ev.Type == EventPut && ev.Value != nil {
				var val T
				if err := unmarshalVal(ev.Value, &val); err != nil {
					logman.Warn("cstore: Watch 事件反序列化失败", "key", ev.Key, "error", err)
				} else {
					te.Value = val
					te.Valid = true
				}
			}
			out <- te
		}
	}()
	return out
}

// Key 返回持有的 key（文件名）。
func (t *TypedStore[T]) Key() string {
	return t.key
}

// Store 返回底层 Store。
func (t *TypedStore[T]) Store() Store {
	return t.store
}

// Close 释放底层 Store 资源。
func (t *TypedStore[T]) Close() error {
	return t.store.Close()
}

// unmarshalVal 将 YAML 反序列化到 v。
// T 为指针类型时自动分配内层对象，避免传 **T 给 go-yaml 导致类型不匹配。
func unmarshalVal[T any](data []byte, v *T) error {
	rv := reflect.ValueOf(v).Elem() // *T → T
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return yaml.Unmarshal(data, rv.Interface())
	}
	return yaml.Unmarshal(data, v)
}
