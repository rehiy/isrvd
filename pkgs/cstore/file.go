package cstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore 基于本地文件系统的配置存储。
// key 为文件名（如 "config.yml"），拼接到 baseDir 后得到完整路径。
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

// newFileStore 创建 FileStore，baseDir 为配置文件所在目录。
func newFileStore(baseDir string) (*FileStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("cstore/file: baseDir 不能为空")
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("cstore/file: 解析路径失败: %w", err)
	}
	return &FileStore{baseDir: abs}, nil
}

func (f *FileStore) path(key string) string {
	return filepath.Join(f.baseDir, key)
}

// Get 读取 key 对应文件内容，文件不存在时返回 nil, nil。
func (f *FileStore) Get(key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cstore/file: 读取 %s 失败: %w", key, err)
	}
	return data, nil
}

// Set 将 value 写入 key 对应的文件，自动创建目录。
func (f *FileStore) Set(key string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p := f.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("cstore/file: 创建目录失败: %w", err)
	}
	if err := os.WriteFile(p, value, 0644); err != nil {
		return fmt.Errorf("cstore/file: 写入 %s 失败: %w", key, err)
	}
	return nil
}

// Watch 文件后端不支持变更监听，返回 nil（select 中永远阻塞，调用方可安全使用）。
func (f *FileStore) Watch(_ context.Context, _ string) <-chan Event {
	return nil
}

func (f *FileStore) Close() error {
	return nil
}
