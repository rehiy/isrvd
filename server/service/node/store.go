package node

import (
	"errors"
	"slices"
	"sync"

	"github.com/rehiy/libgo/strutil"

	"isrvd/pkgs/cstore"
	"isrvd/server/config"
)

const nodeListFile = "node-list.yml"

// nodeStore 带互斥的类型化列表。持久化交给业务数据存储（config.OpenData），
// 与计划任务、SSH 主机一致：后端选择、原子写入与 etcd 同步都由 cstore 负责。
// 数据目录（rootDirectory）随配置重载变化时，下一次访问会使用对应位置，与其他业务数据的重载行为一致。
type nodeStore struct {
	mu    sync.RWMutex
	root  string // 当前数据所在的 rootDirectory
	ts    *cstore.TypedStore[[]*Node]
	items []*Node
}

func newNodeStore() (*nodeStore, error) {
	root := config.Current().Server.RootDirectory
	ts, items, err := openNodes()
	if err != nil {
		return nil, err
	}
	return &nodeStore{root: root, ts: ts, items: items}, nil
}

func openNodes() (*cstore.TypedStore[[]*Node], []*Node, error) {
	ts, err := config.OpenData[[]*Node](nodeListFile)
	if err != nil {
		return nil, nil, err
	}
	items, err := ts.Get()
	if err != nil {
		return nil, nil, err
	}
	return ts, slices.DeleteFunc(items, func(item *Node) bool { return item == nil }), nil
}

// refresh 在 rootDirectory 变化后加载对应位置的数据。打开失败时保留旧数据并只告警一次（重启后生效）。
func (s *nodeStore) refresh() {
	root := config.Current().Server.RootDirectory
	s.mu.RLock()
	same := root == s.root
	s.mu.RUnlock()
	if same {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if root == s.root {
		return
	}
	s.root = root
	ts, items, err := openNodes()
	if err != nil {
		logger.Warn("rootDirectory 已变更但无法打开新位置，继续使用旧位置，重启后生效", "file", nodeListFile, "error", err)
		return
	}
	s.ts, s.items = ts, items
	logger.Info("数据目录已变更，已切换到新位置", "file", nodeListFile, "root", root)
}

// list 返回全部条目副本
func (s *nodeStore) list() []*Node {
	s.refresh()
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Node, len(s.items))
	for i, item := range s.items {
		result[i] = cloneNode(item)
	}
	return result
}

// byID 按 ID 查找节点副本，不存在返回 nil。
func (s *nodeStore) byID(id string) *Node {
	return s.find(func(n *Node) bool {
		return n.ID == id
	})
}

// find 在锁内查找并返回节点副本。
func (s *nodeStore) find(match func(*Node) bool) *Node {
	s.refresh()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.items {
		if match(n) {
			return cloneNode(n)
		}
	}
	return nil
}

// insert 为 item 生成唯一的节点 ID 并写入存储，成功后回填 item.ID。
func (s *nodeStore) insert(item *Node) error {
	var id string
	err := s.mutate(func(items []*Node) ([]*Node, error) {
		for range 16 {
			id = strutil.Rand(10)
			if slices.ContainsFunc(items, func(n *Node) bool { return n.ID == id }) {
				continue
			}
			next := cloneNode(item)
			next.ID = id
			return append(items, next), nil
		}
		return nil, errors.New("node: failed to generate a unique ID after 16 attempts")
	})
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

// update 在锁内读取-修改-写入；fn 返回错误则放弃修改
func (s *nodeStore) update(id string, fn func(*Node) error) (*Node, error) {
	var updated *Node
	err := s.mutate(func(items []*Node) ([]*Node, error) {
		i := slices.IndexFunc(items, func(item *Node) bool {
			return item.ID == id
		})
		if i < 0 {
			return nil, ErrNotFound
		}
		next := cloneNode(items[i])
		if err := fn(next); err != nil {
			return nil, err
		}
		items[i], updated = next, cloneNode(next)
		return items, nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *nodeStore) remove(id string) error {
	return s.mutate(func(items []*Node) ([]*Node, error) {
		i := slices.IndexFunc(items, func(item *Node) bool {
			return item.ID == id
		})
		if i < 0 {
			return nil, ErrNotFound
		}
		return slices.Delete(items, i, i+1), nil
	})
}

// mutate 在锁内基于当前列表的副本生成新列表并持久化；fn 返回错误则放弃修改
func (s *nodeStore) mutate(fn func(items []*Node) ([]*Node, error)) error {
	s.refresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := fn(slices.Clone(s.items))
	if err != nil {
		return err
	}
	if err := s.ts.Set(items); err != nil {
		return err
	}
	s.items = items
	return nil
}

// ─── 辅助函数 ───

func cloneNode(item *Node) *Node {
	c := *item
	c.PortMaps = slices.Clone(item.PortMaps)
	return &c
}
