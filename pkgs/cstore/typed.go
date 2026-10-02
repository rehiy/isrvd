package cstore

import (
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"
)

// TypedStore 在 Store 之上封装了 YAML 序列化/反序列化，
// 业务层直接操作类型化的值，不再感知 []byte 和 yaml 细节。
type TypedStore[T any] struct {
	document *Document
}

// NewTypedWith 基于已打开的 Store 创建 TypedStore，多个 key 可共享同一后端连接。
func NewTypedWith[T any](s Store, key string) *TypedStore[T] {
	return &TypedStore[T]{document: NewDocument(s, key)}
}

// Get 读取并反序列化值。key 不存在时返回零值和 nil error。
func (t *TypedStore[T]) Get() (val T, err error) {
	_, err = t.document.Read(func(data []byte) error {
		if data == nil {
			return nil
		}
		return unmarshalVal(data, &val)
	})
	if err != nil {
		return val, fmt.Errorf("cstore: 加载 %s 失败: %w", t.document.key, err)
	}
	return val, nil
}

// Set 序列化并按最近一次成功加载/提交的基线写入，未加载或冲突时拒绝保存。
func (t *TypedStore[T]) Set(val T) error {
	data, err := yaml.Marshal(val)
	if err != nil {
		return fmt.Errorf("cstore: 序列化 %s 失败: %w", t.document.key, err)
	}
	return t.document.Set(data)
}

// ─── 辅助函数 ───

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
