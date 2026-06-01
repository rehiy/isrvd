// Package jsonl 提供按天滚动的 JSON Lines 文件读写能力。
//
// 写入：长连接 + bufio + 锁外序列化，降低系统调用与锁竞争
// 读取：流式扫描 + gjson 过滤，避免全量加载
//
// 示例：
//
//	store, _ := jsonl.New("/var/log", jsonl.Naming{Prefix: "app", Suffix: ".jsonl"})
//	defer store.Close()
//	store.Append(map[string]any{"ts": time.Now().Unix(), "msg": "hello"})
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
)

// Naming 定义按天滚动的文件命名规则。
// 完整文件名：filepath.Join(Dir, Prefix+Sep+YYYY-MM-DD+Suffix)
type Naming struct {
	Prefix string // 前缀
	Sep    string // 分隔符
	Suffix string // 后缀
}

// FileName 返回指定日期对应的文件名（不含目录）。
func (n Naming) FileName(date string) string {
	if n.Prefix == "" {
		return date + n.Suffix
	}
	return n.Prefix + n.Sep + date + n.Suffix
}

// Store 管理单个目录下按天滚动的 JSONL 文件追加写入。
type Store struct {
	dir    string
	naming Naming
	opts   options

	mu       sync.Mutex
	date     string
	file     *os.File
	bw       *bufio.Writer
	ch       chan []byte
	stopOnce sync.Once
	stopped  chan struct{}
}

// options Store 的可选参数
type options struct {
	bufSize       int
	asyncSize     int
	flushInterval time.Duration
	loc           *time.Location
}

// Option 定义 Store 构造选项
type Option func(*options)

// WithBufferSize 设置缓冲区大小（字节），0 表示直写。
func WithBufferSize(n int) Option {
	return func(o *options) { o.bufSize = n }
}

// WithAsync 启用异步写入，channel 容量为 size。
func WithAsync(size int) Option {
	return func(o *options) { o.asyncSize = size }
}

// WithFlushInterval 设置周期 flush 间隔。
func WithFlushInterval(d time.Duration) Option {
	return func(o *options) { o.flushInterval = d }
}

// WithLocation 指定时区，默认 time.Local。
func WithLocation(loc *time.Location) Option {
	return func(o *options) { o.loc = loc }
}

// New 创建 Store，dir 不存在时自动创建。
func New(dir string, naming Naming, opts ...Option) (*Store, error) {
	o := options{
		bufSize:       4096,
		flushInterval: time.Second,
		loc:           time.Local,
	}
	for _, fn := range opts {
		fn(&o)
	}
	if o.loc == nil {
		o.loc = time.Local
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("jsonl: mkdir %s: %w", dir, err)
	}

	s := &Store{
		dir:     dir,
		naming:  naming,
		opts:    o,
		stopped: make(chan struct{}),
	}

	if o.asyncSize > 0 {
		s.ch = make(chan []byte, o.asyncSize)
		go s.runAsync()
	}
	if o.bufSize > 0 && o.flushInterval > 0 && o.asyncSize == 0 {
		// 同步模式下也提供周期 flush，避免 bufio 数据滞留
		go s.runFlusher()
	}
	return s, nil
}

// FilePath 返回指定日期的完整文件路径
func (s *Store) FilePath(date string) string {
	return filepath.Join(s.dir, s.naming.FileName(date))
}

// Today 返回当前日期（YYYY-MM-DD）。
func (s *Store) Today() string {
	return time.Now().In(s.opts.loc).Format(dateLayout)
}

// Append 序列化并追加一条记录。异步模式入队即返回，同步模式落盘后返回。
func (s *Store) Append(v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("jsonl: marshal: %w", err)
	}
	line = append(line, '\n')

	if s.ch != nil {
		select {
		case s.ch <- line:
			return nil
		default:
			return fmt.Errorf("jsonl: async channel full")
		}
	}
	return s.writeLocked(line)
}

// writeLocked 加锁写入当前文件
func (s *Store) writeLocked(line []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureFileLocked(); err != nil {
		return err
	}

	if s.bw != nil {
		_, err := s.bw.Write(line)
		return err
	}
	_, err := s.file.Write(line)
	return err
}

// ensureFileLocked 确保文件句柄对应今天，必要时轮转。
func (s *Store) ensureFileLocked() error {
	today := s.Today()

	if s.file != nil && s.date == today {
		return nil
	}

	if s.file != nil {
		if s.bw != nil {
			_ = s.bw.Flush()
		}
		_ = s.file.Close()
		s.file = nil
		s.bw = nil
	}

	path := s.FilePath(today)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("jsonl: open %s: %w", path, err)
	}
	s.file = f
	s.date = today
	if s.opts.bufSize > 0 {
		s.bw = bufio.NewWriterSize(f, s.opts.bufSize)
	}
	return nil
}

// Flush 刷盘，仅 bufio 模式有效。
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bw != nil {
		return s.bw.Flush()
	}
	return nil
}

// Close 停止异步写入、刷盘并关闭文件。可安全调用多次。
func (s *Store) Close() error {
	s.stopOnce.Do(func() {
		if s.ch != nil {
			close(s.ch)
			<-s.stopped
		} else {
			close(s.stopped)
		}
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	var flushErr error
	if s.bw != nil {
		flushErr = s.bw.Flush()
		s.bw = nil
	}
	if s.file != nil {
		closeErr := s.file.Close()
		s.file = nil
		if flushErr != nil {
			return flushErr
		}
		return closeErr
	}
	return flushErr
}

// runAsync 后台串行落盘
func (s *Store) runAsync() {
	var ticker *time.Ticker
	var tickC <-chan time.Time

	if s.opts.bufSize > 0 && s.opts.flushInterval > 0 {
		ticker = time.NewTicker(s.opts.flushInterval)
		tickC = ticker.C
		defer ticker.Stop()
	}

	for {
		select {
		case line, ok := <-s.ch:
			if !ok {
				s.handleError(s.Flush())
				close(s.stopped)
				return
			}
			s.handleError(s.writeLocked(line))
		case <-tickC:
			s.handleError(s.Flush())
		}
	}
}

// runFlusher 同步模式定期刷盘
func (s *Store) runFlusher() {
	ticker := time.NewTicker(s.opts.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopped:
			return
		case <-ticker.C:
			s.handleError(s.Flush())
		}
	}
}

// handleError 处理后台错误。
func (s *Store) handleError(err error) {
	if err != nil {
		logman.Error("jsonl: background error", "error", err)
	}
}
