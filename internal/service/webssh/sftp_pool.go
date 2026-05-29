package webssh

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	libwebssh "github.com/rehiy/libgo/webssh"
)

const sftpIdleTimeout = 30 * time.Second

// sftpEntry 连接池中的一个缓存条目
type sftpEntry struct {
	client   *sftp.Client
	lastUsed time.Time
}

// sftpPool 按 hostID 缓存 sftp.Client，空闲超时后自动关闭
type sftpPool struct {
	mu      sync.Mutex
	entries map[string]*sftpEntry
	stopCh  chan struct{}
}

func newSFTPPool() *sftpPool {
	p := &sftpPool{
		entries: make(map[string]*sftpEntry),
		stopCh:  make(chan struct{}),
	}
	go p.evictLoop()
	return p
}

// get 从池中获取连接，若不存在或已失效则新建
func (p *sftpPool) get(hostID string, store *store) (*sftp.Client, error) {
	// 先在锁内取出 client，释放锁后再做网络探活，避免持锁期间阻塞
	p.mu.Lock()
	e, ok := p.entries[hostID]
	p.mu.Unlock()

	if ok {
		// 锁外探活：Getwd 是一次 SFTP 网络请求，不能在持锁时执行
		if _, err := e.client.Getwd(); err == nil {
			p.mu.Lock()
			e.lastUsed = time.Now()
			p.mu.Unlock()
			return e.client, nil
		}
		// 连接已断开，清理
		e.client.Close()
		p.mu.Lock()
		delete(p.entries, hostID)
		p.mu.Unlock()
	}

	// 建立新连接
	client, err := p.dial(hostID, store)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.entries[hostID] = &sftpEntry{client: client, lastUsed: time.Now()}
	p.mu.Unlock()
	return client, nil
}

// invalidate 主动移除某个 hostID 的缓存（连接出错时调用）
func (p *sftpPool) invalidate(hostID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.entries[hostID]; ok {
		e.client.Close()
		delete(p.entries, hostID)
	}
}

// evictLoop 定期清理空闲超时的连接
func (p *sftpPool) evictLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			for id, e := range p.entries {
				if time.Since(e.lastUsed) > sftpIdleTimeout {
					e.client.Close()
					delete(p.entries, id)
				}
			}
			p.mu.Unlock()
		case <-p.stopCh:
			return
		}
	}
}

// close 关闭连接池，释放所有连接
func (p *sftpPool) close() {
	close(p.stopCh)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		e.client.Close()
	}
	p.entries = make(map[string]*sftpEntry)
}

// dial 建立新的 SSH+SFTP 连接
func (p *sftpPool) dial(hostID string, store *store) (*sftp.Client, error) {
	host, err := store.hostGetOption(hostID)
	if err != nil {
		return nil, err
	}
	opt := &libwebssh.SSHClientOption{
		Addr:       host.Addr,
		User:       host.User,
		Password:   host.Password,
		PrivateKey: host.PrivateKey,
	}
	sshClient, err := libwebssh.NewSSHClient(opt)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("SFTP 初始化失败: %w", err)
	}
	return sftpClient, nil
}
