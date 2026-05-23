package cron

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"

	"isrvd/config"
)

// Store 负责计划任务配置和执行日志的文件存储。
type Store struct {
	rootDirectory string
	jobMu         sync.Mutex
	logMu         sync.Mutex
}

// NewStore 创建计划任务文件存储。
func NewStore(rootDirectory string) *Store {
	return &Store{rootDirectory: rootDirectory}
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
		job.WorkDir = config.PathToAbs(s.rootDirectory, job.WorkDir)
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
		job.WorkDir = config.PathToRel(s.rootDirectory, job.WorkDir)
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

// AppendJobLog 将任务执行日志按任务 ID 追加到 logs/cron/{jobID}.log，每行一条 JSON。
func (s *Store) AppendJobLog(entry *JobLog) {
	if entry == nil || entry.JobID == "" {
		return
	}

	s.logMu.Lock()
	defer s.logMu.Unlock()

	path, ok := s.jobLogPath(entry.JobID)
	if !ok {
		logger.Warn("无效的计划任务日志 ID", "id", entry.JobID)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		logger.Warn("无法创建计划任务日志目录", "error", err)
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		logger.Warn("序列化计划任务日志失败", "error", err)
		return
	}
	data = append(data, '\n')

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("无法打开计划任务日志文件", "path", path, "error", err)
		return
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		logger.Warn("写入计划任务日志失败", "path", path, "error", err)
	}
}

// LoadJobLogs 从 logs/cron/{jobID}.log 读取最近的任务执行日志，按时间倒序返回。
func (s *Store) LoadJobLogs(id string, limit int) []*JobLog {
	if limit <= 0 {
		limit = 50
	}

	path, ok := s.jobLogPath(id)
	if !ok {
		return nil
	}

	entries := s.readJobLogFile(path)
	result := make([]*JobLog, 0, limit)
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, entries[i])
	}
	return result
}

func (s *Store) readJobLogFile(path string) []*JobLog {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var entries []*JobLog
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		var entry JobLog
		if json.Unmarshal(scanner.Bytes(), &entry) == nil {
			entries = append(entries, &entry)
		}
	}
	return entries
}

func (s *Store) jobLogPath(id string) (string, bool) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		return "", false
	}
	return filepath.Join(s.rootDirectory, "logs", "cron", id+".log"), true
}
