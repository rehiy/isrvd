package webssh

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/rehiy/libgo/strutil"
	libwebssh "github.com/rehiy/libgo/webssh"

	"isrvd/config"
)

// Host SSH 主机配置
type Host struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Addr        string `yaml:"addr" json:"addr"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password,omitempty" json:"password,omitempty"`
	PrivateKey  string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	Description string `yaml:"description" json:"description"`
}

// HostView 主机视图（密码不回显，仅返回是否已设置）
type HostView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	User        string `json:"user"`
	PasswordSet bool   `json:"passwordSet"`
	PrivateKey  string `json:"privateKey,omitempty"`
	Description string `json:"description"`
}

// toView 将 Host 转为视图（密码不回显）
func (h *Host) toView() *HostView {
	return &HostView{
		ID:          h.ID,
		Name:        h.Name,
		Addr:        h.Addr,
		User:        h.User,
		PasswordSet: h.Password != "",
		PrivateKey:  h.PrivateKey,
		Description: h.Description,
	}
}

// store 负责 WebSSH 主机配置的文件存储
type store struct {
	path  string
	mu    sync.RWMutex
	hosts []*Host
}

// newStore 创建主机配置存储
func newStore() (*store, error) {
	p := filepath.Join(config.Server.RootDirectory, "webssh.yml")
	s := &store{path: p}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load 从文件加载主机列表（文件不存在时初始化为空列表）
func (s *store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.hosts = []*Host{}
			return nil
		}
		return fmt.Errorf("读取 webssh.yml 失败: %w", err)
	}
	var hosts []*Host
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return fmt.Errorf("解析 webssh.yml 失败: %w", err)
	}
	s.hosts = hosts
	return nil
}

// save 将主机列表写入文件
func (s *store) save() error {
	data, err := yaml.Marshal(s.hosts)
	if err != nil {
		return fmt.Errorf("序列化主机配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("写入 webssh.yml 失败: %w", err)
	}
	return nil
}

// hostList 返回所有主机的视图列表（密码不回显）
func (s *store) hostList() []*HostView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	views := make([]*HostView, 0, len(s.hosts))
	for _, h := range s.hosts {
		views = append(views, h.toView())
	}
	return views
}

// hostInspect 返回指定 ID 的主机视图
func (s *store) hostInspect(id string) *HostView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.findByID(id)
	if h == nil {
		return nil
	}
	return h.toView()
}

// hostCreate 新建主机配置
func (s *store) hostCreate(h *Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	h.ID = strutil.NewString()
	s.hosts = append(s.hosts, h)
	return s.save()
}

// hostUpdate 更新主机配置；密码/私钥为空时保留原值
func (s *store) hostUpdate(id string, h *Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.findByID(id)
	if old == nil {
		return fmt.Errorf("主机 %s 不存在", id)
	}
	h.ID = id
	// 密码/私钥为空时保留原值（前端编辑时不回显敏感信息）
	if h.Password == "" {
		h.Password = old.Password
	}
	if h.PrivateKey == "" {
		h.PrivateKey = old.PrivateKey
	}
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
func (s *store) hostGetOption(id string) (*libwebssh.SSHClientOption, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.findByID(id)
	if h == nil {
		return nil, fmt.Errorf("主机 %s 不存在", id)
	}
	return &libwebssh.SSHClientOption{
		Addr:       h.Addr,
		User:       h.User,
		Password:   h.Password,
		PrivateKey: h.PrivateKey,
	}, nil
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
