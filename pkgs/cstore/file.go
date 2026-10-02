package cstore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileLocks 协调同目录的多个存储实例（仅限本进程）。
var fileLocks sync.Map

// FileStore 基于本地文件系统的配置存储。
// key 为文件名（如 "config.yml"），拼接到 baseDir 后得到完整路径。
type FileStore struct {
	baseDir string
	mu      *sync.RWMutex
}

func (f *FileStore) path(key string) string {
	return filepath.Join(f.baseDir, key)
}

// Get 读取 key 对应文件内容，文件不存在时返回 nil, nil。
func (f *FileStore) Get(key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.read(key)
}

func (f *FileStore) read(key string) ([]byte, error) {
	data, err := os.ReadFile(f.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cstore/file: 读取 %s 失败: %w", key, err)
	}
	return data, nil
}

// Set 原子替换文件，保留现有权限；新文件使用 0600，自动创建目录。
func (f *FileStore) Set(key string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.write(key, value)
}

// CheckAndSet 防止本进程中不同 Store 实例覆盖已变更的文件。
func (f *FileStore) CheckAndSet(key string, expected, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	current, err := f.read(key)
	if err != nil {
		return err
	}
	if (current == nil) != (expected == nil) || !bytes.Equal(current, expected) {
		return ErrConflict
	}
	return f.write(key, value)
}

func (f *FileStore) write(key string, value []byte) error {
	p := f.path(key)
	// 保留已有符号链接的写入目标，避免原子替换把链接本身覆盖掉。
	if info, err := os.Lstat(p); err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(p)
		if err != nil {
			return err
		}
		p = target
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	mode := os.FileMode(0600)
	if info, err := os.Stat(p); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(p), ".cstore-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}()
	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if _, err := temp.Write(value); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), p)
}

// Watch 文件后端不支持变更监听，返回 nil（select 中永远阻塞，调用方可安全使用）。
func (f *FileStore) Watch(_ context.Context, _ string) <-chan Event {
	return nil
}

func (f *FileStore) Close() error {
	return nil
}

// ─── 辅助函数 ───

// newFileStore 创建 FileStore，baseDir 为配置文件所在目录。
func newFileStore(baseDir string) (*FileStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("cstore/file: baseDir 不能为空")
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("cstore/file: 解析路径失败: %w", err)
	}
	lock, _ := fileLocks.LoadOrStore(abs, &sync.RWMutex{})
	return &FileStore{baseDir: abs, mu: lock.(*sync.RWMutex)}, nil
}
