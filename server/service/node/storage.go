package node

import (
	"errors"
	"slices"
	"sync"
	"time"

	"isrvd/pkgs/cstore"
	"isrvd/server/config"
)

// 节点状态
const (
	StatusPending  = "pending"  // 已提交注册，等待管理员审批
	StatusApproved = "approved" // 已审批，可连接
	StatusRevoked  = "revoked"  // 已吊销，令牌失效
)

// ErrNotFound 表示节点或注册码不存在。
var ErrNotFound = errors.New("记录不存在")

// Node 受管机记录。令牌与领取密钥只保存 SHA-256 哈希，任何接口都不回显。
type Node struct {
	ID        string    `yaml:"id" json:"id"`                 // 节点 ID（自动生成）
	Name      string    `yaml:"name" json:"name"`             // 节点名称
	Status    string    `yaml:"status" json:"status"`         // pending / approved / revoked
	Hostname  string    `yaml:"hostname" json:"hostname"`     // 注册时上报的主机名
	OS        string    `yaml:"os" json:"os"`                 // 注册时上报的操作系统
	Arch      string    `yaml:"arch" json:"arch"`             // 注册时上报的 CPU 架构
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`   // 注册时间
	ClaimHash string    `yaml:"claimHash,omitempty" json:"-"` // 一次性领取密钥哈希；节点首次成功连接后清除
	TokenHash string    `yaml:"tokenHash,omitempty" json:"-"` // 节点令牌哈希

	// 以下为运行时状态，不持久化
	Online       bool       `yaml:"-" json:"online"`                 // 是否在线
	Latency      int64      `yaml:"-" json:"latency"`                // 隧道往返延迟（毫秒），离线为 0
	AgentVersion string     `yaml:"-" json:"agentVersion,omitempty"` // 节点程序版本
	Compatible   bool       `yaml:"-" json:"compatible"`             // 协议版本是否与中控兼容
	ConnectedAt  *time.Time `yaml:"-" json:"connectedAt,omitempty"`  // 本次连接时间
	RemoteAddr   string     `yaml:"-" json:"remoteAddr,omitempty"`   // 节点连接来源地址
}

// EnrollCode 注册码。使用一次即失效、最长 7 天过期；创始人可在待接入期间随时查看接入命令，
// 所以不能只存哈希，而是与 SSH 凭据一样用 JWT 密钥派生的密钥加密落盘（见 pkgs/secretbox）。
// 长期有效的节点令牌与领取密钥仍然只保存哈希。
type EnrollCode struct {
	ID          string    `yaml:"id" json:"id"`                   // 注册码 ID
	Name        string    `yaml:"name" json:"name"`               // 备注
	Sealed      string    `yaml:"code" json:"-"`                  // 注册码密文（落盘）
	Code        string    `yaml:"-" json:"code"`                  // 注册码明文，由服务从密文解出后填入，不落盘
	AutoApprove bool      `yaml:"autoApprove" json:"autoApprove"` // 使用该码注册的节点是否自动审批
	CreatedBy   string    `yaml:"createdBy" json:"createdBy"`     // 创建人
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`     // 创建时间
	ExpiresAt   time.Time `yaml:"expiresAt" json:"expiresAt"`     // 过期时间
}

// listStore 带互斥的类型化列表。持久化交给业务数据存储（config.OpenData），
// 与计划任务、SSH 主机一致：后端选择、原子写入与 etcd 同步都由 cstore 负责。
// 数据目录（rootDirectory）随配置重载变化时，下一次访问会自动改用新位置，与其他业务数据的重载行为一致。
type listStore[T any] struct {
	mu    sync.RWMutex
	file  string
	root  string // 当前数据所在的 rootDirectory
	ts    *cstore.TypedStore[[]*T]
	items []*T
	idOf  func(*T) string
}

func newListStore[T any](file string, idOf func(*T) string) (*listStore[T], error) {
	root := config.Current().Server.RootDirectory
	ts, items, err := openList[T](file)
	if err != nil {
		return nil, err
	}
	return &listStore[T]{file: file, root: root, ts: ts, items: items, idOf: idOf}, nil
}

func openList[T any](file string) (*cstore.TypedStore[[]*T], []*T, error) {
	ts, err := config.OpenData[[]*T](file)
	if err != nil {
		return nil, nil, err
	}
	items, err := ts.Get()
	if err != nil {
		return nil, nil, err
	}
	return ts, slices.DeleteFunc(items, func(item *T) bool { return item == nil }), nil
}

// refresh 在 rootDirectory 变化后改用新位置的数据。打开失败时保留旧数据并只告警一次（重启后生效）。
func (s *listStore[T]) refresh() {
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
	ts, items, err := openList[T](s.file)
	if err != nil {
		logger.Warn("rootDirectory 已变更但无法打开新位置，继续使用旧位置，重启后生效", "file", s.file, "error", err)
		return
	}
	s.ts, s.items = ts, items
	logger.Info("数据目录已变更，已切换到新位置", "file", s.file, "root", root)
}

// list 返回全部条目副本
func (s *listStore[T]) list() []*T {
	s.refresh()
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*T, len(s.items))
	for i, item := range s.items {
		result[i] = cloneItem(item)
	}
	return result
}

// get 返回指定条目副本，不存在返回 nil
func (s *listStore[T]) get(id string) *T {
	s.refresh()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i := s.indexOf(id); i >= 0 {
		return cloneItem(s.items[i])
	}
	return nil
}

func (s *listStore[T]) insert(item *T) error {
	s.refresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commit(append(slices.Clone(s.items), cloneItem(item)))
}

// update 在锁内读取-修改-写入；fn 返回错误则放弃修改
func (s *listStore[T]) update(id string, fn func(*T) error) error {
	s.refresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOf(id)
	if i < 0 {
		return ErrNotFound
	}
	next := cloneItem(s.items[i])
	if err := fn(next); err != nil {
		return err
	}
	items := slices.Clone(s.items)
	items[i] = next
	return s.commit(items)
}

func (s *listStore[T]) remove(id string) error {
	s.refresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOf(id)
	if i < 0 {
		return ErrNotFound
	}
	return s.commit(slices.Delete(slices.Clone(s.items), i, i+1))
}

// removeIf 删除满足条件的全部条目
func (s *listStore[T]) removeIf(pred func(*T) bool) error {
	s.refresh()
	s.mu.Lock()
	defer s.mu.Unlock()
	items := slices.DeleteFunc(slices.Clone(s.items), pred)
	if len(items) == len(s.items) {
		return nil
	}
	return s.commit(items)
}

// commit 持久化后替换内存数据（调用方须持锁）
func (s *listStore[T]) commit(items []*T) error {
	if items == nil {
		items = []*T{}
	}
	if err := s.ts.Set(items); err != nil {
		return err
	}
	s.items = items
	return nil
}

func (s *listStore[T]) indexOf(id string) int {
	return slices.IndexFunc(s.items, func(item *T) bool { return s.idOf(item) == id })
}

// ─── 辅助函数 ───

func cloneItem[T any](item *T) *T {
	c := *item
	return &c
}
