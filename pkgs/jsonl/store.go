// Package jsonl 提供按天滚动的 JSON Lines (.jsonl) 文件读写能力。
//
// 设计目标：
//   - 写入：长开句柄 + bufio.Writer + 锁外序列化，降低 syscall 与锁竞争
//   - 读取：bufio.Scanner 流式扫描 + gjson 字段过滤，避免全文件载入与重复反序列化
//   - 支持自定义文件命名（前缀/分隔符/后缀），适配监控、审计等不同场景
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

// Naming 描述按天滚动的文件命名规则。
//
// 完整文件名为：filepath.Join(Dir, Prefix+Sep+YYYY-MM-DD+Suffix)
//
// 例如：
//   - 监控：Naming{Prefix:"host", Sep:"_", Suffix:".jsonl"} -> host_2026-05-28.jsonl
//   - 审计：Naming{Prefix:"",     Sep:"",  Suffix:".jsonl"} -> 2026-05-28.jsonl
type Naming struct {
	Prefix string
	Sep    string
	Suffix string
}

// FileName 返回指定日期对应的文件名（不含目录）。
func (n Naming) FileName(date string) string {
	if n.Prefix == "" {
		return date + n.Suffix
	}
	return n.Prefix + n.Sep + date + n.Suffix
}

// Store 管理单个目录 + 单个 Naming 下的按天追加写。
//
// 多个 Store 实例可指向同一目录但使用不同 Prefix；同一目录+Prefix 的实例
// 应在调用方层面单例化（监控/审计本身就是单例服务，无需包内全局表）。
type Store struct {
	dir    string
	naming Naming
	opts   options

	mu   sync.Mutex
	date string // 当前文件对应日期（YYYY-MM-DD）
	file *os.File
	bw   *bufio.Writer

	// 异步写入相关（仅 async 模式启用）
	ch       chan []byte
	stopOnce sync.Once
	stopped  chan struct{}
}

// options Store 的可选参数集合
type options struct {
	bufSize       int           // bufio.Writer 缓冲区大小，0 表示不使用 bufio
	asyncSize     int           // 异步通道大小，0 表示同步写
	flushInterval time.Duration // 周期 flush 间隔，仅 bufSize>0 时生效
	loc           *time.Location
}

// Option Store 构造选项
type Option func(*options)

// WithBufferSize 设置 bufio.Writer 缓冲区大小（字节）。0 表示直写文件。
func WithBufferSize(n int) Option { return func(o *options) { o.bufSize = n } }

// WithAsync 启用异步写入，channel 容量为 size。
// 异步模式下 Append 仅做序列化与入队，落盘由后台 goroutine 串行执行。
func WithAsync(size int) Option { return func(o *options) { o.asyncSize = size } }

// WithFlushInterval 设置周期 flush 间隔；仅在启用 bufio 时有意义。
func WithFlushInterval(d time.Duration) Option { return func(o *options) { o.flushInterval = d } }

// WithLocation 指定日期切分的时区，默认 time.Local
func WithLocation(loc *time.Location) Option { return func(o *options) { o.loc = loc } }

// New 创建 Store。dir 不存在时自动创建。
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

// FilePath 返回指定日期对应的完整路径
func (s *Store) FilePath(date string) string {
	return filepath.Join(s.dir, s.naming.FileName(date))
}

// Today 返回 Store 时区下的当前日期字符串（YYYY-MM-DD）。
func (s *Store) Today() string {
	return time.Now().In(s.opts.loc).Format(dateLayout)
}

// Append 序列化后追加一条记录。
// 异步模式下序列化在调用方 goroutine 完成（避免阻塞业务线程过久），
// 入队后立即返回；同步模式下落盘成功后返回。
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
			logman.Warn("jsonl: async channel full, drop record", "dir", s.dir, "prefix", s.naming.Prefix)
			return fmt.Errorf("jsonl: async channel full")
		}
	}
	return s.writeLocked(line)
}

// writeLocked 加锁直写当前文件
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

// ensureFileLocked 确保 s.file 对应今天，必要时轮转
func (s *Store) ensureFileLocked() error {
	today := s.Today()
	if s.file != nil && s.date == today {
		return nil
	}
	// 轮转：刷盘并关闭旧文件
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

// Flush 主动将缓冲区写入磁盘
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bw != nil {
		return s.bw.Flush()
	}
	return nil
}

// Close 停止异步 goroutine、刷盘并关闭文件句柄
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
	if s.bw != nil {
		_ = s.bw.Flush()
		s.bw = nil
	}
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
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
				_ = s.Flush()
				close(s.stopped)
				return
			}
			if err := s.writeLocked(line); err != nil {
				logman.Warn("jsonl: async write failed", "dir", s.dir, "prefix", s.naming.Prefix, "error", err)
			}
		case <-tickC:
			_ = s.Flush()
		}
	}
}

// runFlusher 同步模式下的周期 flush
func (s *Store) runFlusher() {
	ticker := time.NewTicker(s.opts.flushInterval)
	defer ticker.Stop()
	for range ticker.C {
		// Close 后无需再 flush；通过 stopped 判断
		select {
		case <-s.stopped:
			return
		default:
		}
		_ = s.Flush()
	}
}
