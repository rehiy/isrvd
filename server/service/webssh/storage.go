package webssh

import (
	"crypto/cipher"
	"fmt"
	"slices"
	"sync"

	"github.com/rehiy/libgo/strutil"

	"isrvd/pkgs/cstore"
	"isrvd/server/config"
)

// storeItem 可存储条目：暴露 ID 与密码/私钥字段
type storeItem[T any] interface {
	*T
	fields() (id, password, privateKey *string)
}

// itemStore 基于 YAML 文件的条目存储；内存保存明文，落盘时加密密码/私钥
type itemStore[T any, P storeItem[T]] struct {
	ts      *cstore.TypedStore[[]P]
	aead    cipher.AEAD
	label   string            // 条目名称，用于错误提示
	prepare func(item, old P) // 保存前调整字段，新建时 old 为 nil
	items   []P
	mu      sync.RWMutex
}

func (s *itemStore[T, P]) list() []P {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]P, len(s.items))
	for i, item := range s.items {
		result[i] = clone[T](item)
	}
	return result
}

// get 返回指定 ID 的条目副本（含敏感信息，仅内部使用）
func (s *itemStore[T, P]) get(id string) P {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i := s.indexOf(id); i >= 0 {
		return clone[T](s.items[i])
	}
	return nil
}

// create 新建条目，保存后的结果回写到 item
func (s *itemStore[T, P]) create(item P) error {
	next := clone[T](item)
	id, _, _ := next.fields()
	*id = strutil.NewString()
	s.prepare(next, nil)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.commit(append(slices.Clone(s.items), next)); err != nil {
		return err
	}
	*item = *next
	return nil
}

// update 更新条目，保存后的结果回写到 item
func (s *itemStore[T, P]) update(id string, item P) error {
	next := clone[T](item)
	nextID, _, _ := next.fields()
	*nextID = id

	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOf(id)
	if i < 0 {
		return fmt.Errorf("%s %s 不存在", s.label, id)
	}
	s.prepare(next, s.items[i])
	items := slices.Clone(s.items)
	items[i] = next
	if err := s.commit(items); err != nil {
		return err
	}
	*item = *next
	return nil
}

func (s *itemStore[T, P]) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOf(id)
	if i < 0 {
		return fmt.Errorf("%s %s 不存在", s.label, id)
	}
	return s.commit(slices.Delete(slices.Clone(s.items), i, i+1))
}

// commit 加密密码/私钥后持久化，成功后替换内存数据（调用方须持锁）
func (s *itemStore[T, P]) commit(items []P) error {
	sealed := make([]P, len(items))
	for i, item := range items {
		sealed[i] = clone[T](item)
		_, password, privateKey := sealed[i].fields()
		*password, *privateKey = sealSecret(s.aead, *password), sealSecret(s.aead, *privateKey)
	}
	if err := s.ts.Set(sealed); err != nil {
		return err
	}
	s.items = items
	return nil
}

// indexOf 按 ID 查找下标（调用方须持锁）
func (s *itemStore[T, P]) indexOf(id string) int {
	return slices.IndexFunc(s.items, func(item P) bool {
		itemID, _, _ := item.fields()
		return *itemID == id
	})
}

// ─── 辅助函数 ───

// newItemStore 从业务数据存储加载条目并解密密码/私钥
func newItemStore[T any, P storeItem[T]](file, label string, aead cipher.AEAD, prepare func(item, old P)) (*itemStore[T, P], error) {
	ts, err := config.OpenData[[]P](file)
	if err != nil {
		return nil, err
	}
	items, err := ts.Get()
	if err != nil {
		return nil, err
	}
	items = slices.DeleteFunc(items, func(item P) bool { return item == nil })
	for _, item := range items {
		_, password, privateKey := item.fields()
		if *password, err = openSecret(aead, *password); err != nil {
			return nil, err
		}
		if *privateKey, err = openSecret(aead, *privateKey); err != nil {
			return nil, err
		}
	}
	return &itemStore[T, P]{ts: ts, aead: aead, label: label, prepare: prepare, items: items}, nil
}

// keepSecrets 填了私钥则清空密码，填了密码则清空私钥，都未填则沿用旧值
func keepSecrets[T any, P storeItem[T]](item, old P) {
	_, password, privateKey := item.fields()
	switch {
	case *privateKey != "":
		*password = ""
	case *password != "":
		*privateKey = ""
	case old != nil:
		_, oldPassword, oldPrivateKey := old.fields()
		*password, *privateKey = *oldPassword, *oldPrivateKey
	}
}

func clone[T any, P storeItem[T]](item P) P {
	c := *item
	return &c
}
