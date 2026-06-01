package cron

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goccy/go-yaml"

	"isrvd/config"
	"isrvd/pkgs/jsonl"
)

const (
	// cronLogDir 任务执行日志的子目录（位于 rootDirectory 下）
	cronLogDir = "logs/cron"
	// cronLogSuffix 当前日志文件后缀
	cronLogSuffix = ".jsonl"
	// cronLogRetainDays 日志保留天数
	cronLogRetainDays = 3
	// cronLogChannel 异步写入队列长度
	cronLogChannel = 256
)

// Store 负责计划任务配置和执行日志的文件存储。
//
// 任务配置：rootDirectory/cron.yml （yaml）
// 执行日志：rootDirectory/logs/cron/YYYY-MM-DD.jsonl （所有任务合并、按天滚动）
type Store struct {
	rootDirectory string
	jobMu         sync.Mutex

	logStore *jsonl.Store
}

// NewStore 创建计划任务文件存储。
func NewStore() *Store {
	s := &Store{
		rootDirectory: config.Server.RootDirectory,
	}

	dir := s.logDir()
	store, err := jsonl.New(
		dir,
		cronLogNaming(),
		jsonl.WithBufferSize(4096),
		jsonl.WithFlushInterval(time.Second),
		jsonl.WithAsync(cronLogChannel),
	)
	if err != nil {
		logger.Warn("Cron log store init failed", "dir", dir, "error", err)
	} else {
		s.logStore = store
	}
	return s
}

// LoadJobs 从 cron.yml 加载任务列表。
// 读取时将 rootDirectory 内的 WorkDir 转为绝对路径。
func (s *Store) LoadJobs() ([]*Job, error) {
	data, err := os.ReadFile(filepath.Join(s.rootDirectory, "cron.yml"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var jobs []*Job
	if err := yaml.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}

	for _, job := range jobs {
		job.WorkDir = config.PathToAbs(job.WorkDir, s.rootDirectory)
	}

	return jobs, nil
}

// SaveJobs 将任务列表写入 cron.yml。
// 对 jobs 做深拷贝，对副本还原相对路径后序列化，不影响原对象。
func (s *Store) SaveJobs(jobs []*Job) error {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()

	// 深拷贝：序列化再反序列化，得到独立副本
	buf, err := yaml.Marshal(jobs)
	if err != nil {
		return err
	}
	var copy []*Job
	if err := yaml.Unmarshal(buf, &copy); err != nil {
		return err
	}

	// 对副本做路径还原
	for _, job := range copy {
		job.WorkDir = config.PathToRel(job.WorkDir, s.rootDirectory)
	}

	data, err := yaml.Marshal(copy)
	if err != nil {
		return err
	}

	path := filepath.Join(s.rootDirectory, "cron.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	logger.Debug("Save cron jobs", "path", path, "count", len(jobs))
	return os.WriteFile(path, data, 0644)
}

// AppendJobLog 将任务执行日志追加到 logs/cron/YYYY-MM-DD.jsonl，所有任务合并。
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

// LoadJobLogs 按 jobID 倒序读取最近的执行日志（最新在前）。
// 实现：从今日文件起，逐天向前回扫并仅保留 jobId==id 的行；
// 直到凑够 limit 条或回扫到超过 retainDays 范围，避免把整天的所有任务日志加载到内存。
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
	if err := jsonl.CleanOlderThan(s.logDir(), cronLogNaming(), cronLogRetainDays); err != nil {
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

// cronLogNaming cron 日志文件命名规则：YYYY-MM-DD.jsonl（无前缀）
func cronLogNaming() jsonl.Naming {
	return jsonl.Naming{Prefix: "", Sep: "", Suffix: cronLogSuffix}
}

// logDir 返回日志目录的绝对路径
func (s *Store) logDir() string {
	return filepath.Join(s.rootDirectory, cronLogDir)
}
