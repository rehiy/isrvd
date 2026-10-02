package cstore

import (
	"bytes"
	"errors"
	"sync"
)

var (
	ErrConflict  = errors.New("存储内容已变更，请重新加载后重试")
	ErrNotLoaded = errors.New("存储尚未成功加载，禁止覆盖写入")
)

// Document 维护最近一次成功加载/提交的原始基线。
// 调用方仍负责串行化业务读改写；加载失败后禁止提交，提交失败保留旧基线。
// 不自动重试业务操作；提交保障取决于底层 Store。
type Document struct {
	store    Store
	key      string
	mu       sync.Mutex
	baseline []byte
	loaded   bool
}

func NewDocument(store Store, key string) *Document {
	return &Document{store: store, key: key}
}

// Read 读取并验证文档；validate 在缺失时也会收到 nil。
func (d *Document) Read(validate func([]byte) error) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.loaded = false
	data, err := d.store.Get(d.key)
	if err != nil {
		return nil, err
	}
	if validate != nil {
		if err := validate(data); err != nil {
			return nil, err
		}
	}
	d.baseline = bytes.Clone(data)
	d.loaded = true
	return data, nil
}

// Set 仅在持久化内容仍与加载基线一致时提交。
func (d *Document) Set(data []byte) error {
	// Set(nil) 仍写入一个存在的空文档，不能把它记成缺失基线。
	if data == nil {
		data = []byte{}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.loaded {
		return ErrNotLoaded
	}
	if err := d.store.CheckAndSet(d.key, d.baseline, data); err != nil {
		return err
	}
	d.baseline = bytes.Clone(data)
	return nil
}
