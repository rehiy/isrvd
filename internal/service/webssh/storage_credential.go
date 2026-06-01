package webssh

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/strutil"

	"isrvd/config"
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

// toView 将 Credential 转为视图（密码/私钥不回显）
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

// credentialStore 负责 Credential 的文件存储
type credentialStore struct {
	path  string
	mu    sync.RWMutex
	items []*Credential
}

// newCredentialStore 创建凭据存储
func newCredentialStore() (*credentialStore, error) {
	p := filepath.Join(config.Server.RootDirectory, "webssh-credentials.yml")
	s := &credentialStore{path: p}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load 从文件加载凭据列表（文件不存在时初始化为空列表）
func (s *credentialStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.items = []*Credential{}
			return nil
		}
		return fmt.Errorf("读取 webssh-credentials.yml 失败: %w", err)
	}
	var items []*Credential
	if err := yaml.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("解析 webssh-credentials.yml 失败: %w", err)
	}
	s.items = items
	return nil
}

// save 将凭据列表写入文件
func (s *credentialStore) save() error {
	data, err := yaml.Marshal(s.items)
	if err != nil {
		return fmt.Errorf("序列化凭据配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("写入 webssh-credentials.yml 失败: %w", err)
	}
	return nil
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
