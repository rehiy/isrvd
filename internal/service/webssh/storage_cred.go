package webssh

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rehiy/libgo/strutil"

	"isrvd/config"
	"isrvd/pkgs/cstore"
)

// Credential SSH 认证凭据（可被多台主机复用）
type Credential struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password,omitempty" json:"password,omitempty"`
	PrivateKey  string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
}

// CredentialView 凭据视图（密码不回显）
type CredentialView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
	AuthType    string `json:"authType"` // "password" | "privateKey" | ""
}

func (c *Credential) toView() *CredentialView {
	cv := &CredentialView{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		User:        c.User,
	}
	if c.PrivateKey != "" {
		cv.AuthType = "privateKey"
	} else if c.Password != "" {
		cv.AuthType = "password"
	}
	return cv
}

// credentialStore 负责 Credential 的存储
type credentialStore struct {
	ts    *cstore.TypedStore[[]*Credential]
	items []*Credential
	mu    sync.RWMutex
}

// newCredentialStore 创建凭据存储
func newCredentialStore() (*credentialStore, error) {
	rootDir := config.Server.RootDirectory
	const key = "webssh-cred.yml"

	// 自动迁移：旧文件存在且新文件不存在时，重命名旧文件
	newPath := filepath.Join(rootDir, key)
	oldPath := filepath.Join(rootDir, "webssh-credentials.yml")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Rename(oldPath, newPath)
		}
	}

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

// list 返回所有凭据的视图列表
func (s *credentialStore) list() []*CredentialView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	views := make([]*CredentialView, 0, len(s.items))
	for _, c := range s.items {
		views = append(views, c.toView())
	}
	return views
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
