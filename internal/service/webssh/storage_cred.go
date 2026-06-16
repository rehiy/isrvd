package webssh

import (
	"fmt"
	"sync"

	"github.com/rehiy/libgo/strutil"

	"isrvd/config"
	"isrvd/pkgs/cstore"
)

// Credential SSH 认证凭据（可被多台主机复用）
type Credential struct {
	ID          string `yaml:"id" json:"id"`                                 // 凭据 ID（自动生成）
	Name        string `yaml:"name" json:"name"`                             // 凭据名称
	Description string `yaml:"description" json:"description"`               // 凭据描述
	User        string `yaml:"user" json:"user"`                             // 用户名
	AuthType    string `yaml:"authType,omitempty" json:"authType,omitempty"` // 认证类型："password" | "privateKey" | ""
	Password    string `yaml:"password,omitempty" json:"-"`                  // 密码（不序列化到 JSON）
	PrivateKey  string `yaml:"privateKey,omitempty" json:"-"`                // 私钥（不序列化到 JSON）
}

// setAuthType 根据当前认证字段计算并设置 AuthType
func (c *Credential) setAuthType() {
	if c.PrivateKey != "" {
		c.AuthType = "privateKey"
	} else if c.Password != "" {
		c.AuthType = "password"
	} else {
		c.AuthType = ""
	}
}

// credentialStore 负责 Credential 的存储
type credentialStore struct {
	ts    *cstore.TypedStore[[]*Credential] // 类型化存储实例
	items []*Credential                     // 内存中的凭据列表
	mu    sync.RWMutex                      // 保护 items 的并发访问
}

// newCredentialStore 创建凭据存储
func newCredentialStore() (*credentialStore, error) {
	rootDir := config.Server.RootDirectory
	const key = "webssh-cred.yml"

	ts, err := cstore.NewTyped[[]*Credential](rootDir, key)
	if err != nil {
		return nil, err
	}
	items, err := ts.Get()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*Credential{}
	}
	return &credentialStore{ts: ts, items: items}, nil
}

// save 将凭据列表写入存储
func (s *credentialStore) save() error {
	return s.ts.Set(s.items)
}

// list 返回所有凭据列表
func (s *credentialStore) list() []*Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Credential, len(s.items))
	copy(result, s.items)
	return result
}

// get 返回指定 ID 的凭据（含敏感信息，仅内部使用）
func (s *credentialStore) get(id string) *Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.items {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// create 新建凭据
func (s *credentialStore) create(c *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.ID = strutil.NewString()
	c.setAuthType()
	s.items = append(s.items, c)
	return s.save()
}

// update 更新凭据
func (s *credentialStore) update(id string, c *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexOf(id)
	if idx < 0 {
		return fmt.Errorf("凭据 %s 不存在", id)
	}
	c.ID = id
	c.setAuthType()
	s.items[idx] = c
	return s.save()
}

// delete 删除凭据
func (s *credentialStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexOf(id)
	if idx < 0 {
		return fmt.Errorf("凭据 %s 不存在", id)
	}
	s.items = append(s.items[:idx], s.items[idx+1:]...)
	return s.save()
}

// indexOf 按 ID 查找凭据下标（调用方须持锁）
func (s *credentialStore) indexOf(id string) int {
	for i, c := range s.items {
		if c.ID == id {
			return i
		}
	}
	return -1
}
