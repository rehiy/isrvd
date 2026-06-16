package webssh

import (
	"fmt"
	"sync"

	"github.com/rehiy/libgo/strutil"
	libWebSSH "github.com/rehiy/libgo/webssh"

	"isrvd/config"
	"isrvd/pkgs/cstore"
)

// Host SSH 主机配置
type Host struct {
	ID             string `yaml:"id" json:"id"`                                         // 主机 ID（自动生成）
	Name           string `yaml:"name" json:"name"`                                     // 主机名称
	Addr           string `yaml:"addr" json:"addr"`                                     // 主机地址（host:port）
	CredentialID   string `yaml:"credentialId,omitempty" json:"credentialId,omitempty"` // 引用的凭据 ID
	CredentialName string `yaml:"-" json:"credentialName,omitempty"`                    // 凭据名称（只读，展示用）
	User           string `yaml:"user" json:"user"`                                     // 用户名（独立认证模式）
	Password       string `yaml:"password,omitempty" json:"-"`                          // 密码（仅独立认证模式，不序列化到 JSON）
	PrivateKey     string `yaml:"privateKey,omitempty" json:"-"`                        // 私钥（仅独立认证模式，不序列化到 JSON）
	Description    string `yaml:"description" json:"description"`                       // 主机描述
}

// store 负责 WebSSH 主机配置的存储
type store struct {
	ts    *cstore.TypedStore[[]*Host] // 类型化存储实例
	hosts []*Host                     // 内存中的主机列表
	mu    sync.RWMutex                // 保护 hosts 的并发访问
}

// newHostStore 创建主机配置存储
func newHostStore() (*store, error) {
	rootDir := config.Server.RootDirectory
	const key = "webssh-host.yml"

	ts, err := cstore.NewTyped[[]*Host](rootDir, key)
	if err != nil {
		return nil, err
	}
	hosts, err := ts.Get()
	if err != nil {
		return nil, err
	}
	if hosts == nil {
		hosts = []*Host{}
	}
	return &store{ts: ts, hosts: hosts}, nil
}

// save 将主机列表写入存储
func (s *store) save() error {
	return s.ts.Set(s.hosts)
}

// hostList 返回所有主机列表（密码/私钥不序列化）
func (s *store) hostList() []*Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Host, len(s.hosts))
	copy(result, s.hosts)
	return result
}

// hostInspect 返回指定 ID 的主机
func (s *store) hostInspect(id string) *Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.findByID(id)
}

// hostCreate 新建主机配置
func (s *store) hostCreate(h *Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	h.ID = strutil.NewString()
	s.hosts = append(s.hosts, h)
	return s.save()
}

// hostUpdate 更新主机配置
func (s *store) hostUpdate(id string, h *Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.findByID(id)
	if old == nil {
		return fmt.Errorf("主机 %s 不存在", id)
	}
	h.ID = id
	*old = *h
	return s.save()
}

// hostDelete 删除主机配置
func (s *store) hostDelete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexByID(id)
	if idx < 0 {
		return fmt.Errorf("主机 %s 不存在", id)
	}
	s.hosts = append(s.hosts[:idx], s.hosts[idx+1:]...)
	return s.save()
}

// hostGetOption 获取指定 ID 主机的 SSH 连接配置
// 如果主机绑定了凭据，优先使用凭据中的认证信息
func (s *store) hostGetOption(id string, credStore *credentialStore) (*libWebSSH.SSHClientOption, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.findByID(id)
	if h == nil {
		return nil, fmt.Errorf("主机 %s 不存在", id)
	}
	opt := &libWebSSH.SSHClientOption{
		Addr:       h.Addr,
		User:       h.User,
		Password:   h.Password,
		PrivateKey: h.PrivateKey,
	}
	if h.CredentialID != "" && credStore != nil {
		c := credStore.get(h.CredentialID)
		if c != nil {
			opt.User = c.User
			opt.Password = c.Password
			opt.PrivateKey = c.PrivateKey
		}
	}
	return opt, nil
}

// findByID 按 ID 查找主机（调用方须持锁）
func (s *store) findByID(id string) *Host {
	for _, h := range s.hosts {
		if h.ID == id {
			return h
		}
	}
	return nil
}

// indexByID 按 ID 查找主机下标（调用方须持锁）
func (s *store) indexByID(id string) int {
	for i, h := range s.hosts {
		if h.ID == id {
			return i
		}
	}
	return -1
}
