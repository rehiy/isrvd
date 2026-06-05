package cron

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/rehiy/libgo/jsonl"

	"isrvd/config"
	"isrvd/pkgs/cstore"
)

const (
	// cronLogRetainDays 日志保留天数
	cronLogRetainDays = 3
	// cronLogChannel 异步写入队列长度
	cronLogChannel = 256
)

// Store 负责计划任务配置和执行日志的存储。
//
// 任务配置：rootDirectory/cron.yml （yaml）
// 执行日志：rootDirectory/cron/YYYY-MM-DD.jsonl （所有任务合并、按天滚动）
type Store struct {
	ts      *cstore.TypedStore[[]*Job]
	dataDir string // 日志目录绝对路径
	jobMu   sync.Mutex

	logStore *jsonl.Store
}

// NewStore 创建计划任务存储。
func NewStore() *Store {
	rootDir := config.Server.RootDirectory
	ts, err := cstore.NewTyped[[]*Job](rootDir, "cron.yml")
	if err != nil {
		logger.Warn("Cron config store init failed", "dir", rootDir, "error", err)
	}
	s := &Store{
		ts:      ts,
		dataDir: filepath.Join(rootDir, "cron"),
	}

	store, err := jsonl.New(
		s.dataDir,
		cronLogNaming(),
		jsonl.WithBufferSize(4096),
		jsonl.WithFlushInterval(time.Second),
		jsonl.WithAsync(cronLogChannel),
	)
	if err != nil {
		logger.Warn("Cron log store init failed", "dir", s.dataDir, "error", err)
	} else {
		s.logStore = store
	}
	return s
}

// LoadJobs 从 cron.yml 加载任务列表，WorkDir 转为绝对路径。
func (s *Store) LoadJobs() ([]*Job, error) {
	if s.ts == nil {
		return nil, fmt.Errorf("cron: 配置存储未初始化")
	}
	raw, err := s.ts.Get()
	if err != nil || raw == nil {
		return nil, err
	}
	rootDir := config.Server.RootDirectory
	jobs := make([]*Job, len(raw))
	for i, job := range raw {
		cp := *job
		cp.WorkDir = config.PathToAbs(job.WorkDir, rootDir)
		jobs[i] = &cp
	}
	return jobs, nil
}

// SaveJobs 将任务列表写入 cron.yml，WorkDir 还原为相对路径。
func (s *Store) SaveJobs(jobs []*Job) error {
	if s.ts == nil {
		return fmt.Errorf("cron: 配置存储未初始化")
	}

	// 构造副本并还原相对路径，锁外执行，避免序列化期间长时间持锁
	rootDir := config.Server.RootDirectory
	snapshot := make([]*Job, len(jobs))
	for i, job := range jobs {
		cp := *job
		cp.WorkDir = config.PathToRel(job.WorkDir, rootDir)
		snapshot[i] = &cp
	}

	s.jobMu.Lock()
	defer s.jobMu.Unlock()

	logger.Debug("Save cron jobs", "key", "cron.yml", "count", len(jobs))
	return s.ts.Set(snapshot)
}

// AppendJobLog 将任务执行日志追加到 cron/YYYY-MM-DD.jsonl。
func (s *Store) AppendJobLog(entry *JobLog) {
	if entry == nil || entry.JobID == "" {
		return
	}
	if s.logStore == nil {
		return
	}
	if err := s.logStore.Append(entry); err != nil {
		logger.Warn("写入计划任务日志失败", "error", err)
	}
}

// LoadJobLogs 按 jobID 倒序读取最近 limit 条执行日志。
func (s *Store) LoadJobLogs(id string, limit int) []*JobLog {
	if id == "" || limit <= 0 {
		return nil
	}
	if s.logStore == nil {
		return nil
	}

	filter := jsonl.StrEq("jobId", id)
	entries, err := jsonl.DecodeTailDays[JobLog](s.logStore, cronLogRetainDays, limit, filter)
	if err != nil {
		logger.Warn("读取计划任务日志失败", "error", err)
		return nil
	}
	result := make([]*JobLog, len(entries))
	for i := range entries {
		result[i] = &entries[i]
	}
	return result
}

// CleanOld 清理超过保留天数的日志文件
func (s *Store) CleanOld() {
	if err := jsonl.CleanOlderThan(s.dataDir, cronLogNaming(), cronLogRetainDays); err != nil {
		logger.Warn("清理计划任务日志失败", "error", err)
	}
}

// Close 关闭日志文件句柄并刷盘
func (s *Store) Close() error {
	if s.logStore == nil {
		return nil
	}
	return s.logStore.Close()
}

// cronLogNaming 日志文件命名规则：YYYY-MM-DD.jsonl
func cronLogNaming() jsonl.Naming {
	return jsonl.Naming{Suffix: ".jsonl"}
}
