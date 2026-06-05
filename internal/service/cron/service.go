// Package cron 计划任务业务服务
package cron

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/libgo/command"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/signal"
	"github.com/rehiy/libgo/strutil"
	cronlib "github.com/robfig/cron/v3"

	"isrvd/pkgs/docker"
)

// nextCronCleanTime 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextCronCleanTime() time.Time {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 5, 0, 0, tomorrow.Location())
}

// logger 为 cron 包创建带名称的 logger
var logger = logman.Named("cron")

// TypeInfo 脚本类型描述
type TypeInfo struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Job 计划任务业务类型
type Job struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Schedule    string `yaml:"schedule" json:"schedule"`
	Type        string `yaml:"type" json:"type"` // SHELL | EXEC | BAT | POWERSHELL | DOCKER_TMP | DOCKER_CTR
	Content     string `yaml:"content" json:"content"`
	WorkDir     string `yaml:"workDir" json:"workDir"`
	Image       string `yaml:"image,omitempty" json:"image,omitempty"`         // DOCKER_TMP：镜像名
	Container   string `yaml:"container,omitempty" json:"container,omitempty"` // DOCKER_CTR：目标容器名
	Volumes     string `yaml:"volumes,omitempty" json:"volumes,omitempty"`     // DOCKER_TMP：额外挂载，格式：/host:/container[:ro]，换行分隔
	Timeout     uint   `yaml:"timeout" json:"timeout"`                         // 秒，0 表示不限制
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Description string `yaml:"description" json:"description"`
}

// JobUpsertRequest 创建/更新任务请求（由 server 层传入，service 层负责构建 Job）
type JobUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Schedule    string `json:"schedule" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Content     string `json:"content" binding:"required"`
	WorkDir     string `json:"workDir"`
	Image       string `json:"image"`
	Container   string `json:"container"`
	Volumes     string `json:"volumes"`
	Timeout     uint   `json:"timeout"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// JobDetail 任务详情（含运行时调度状态）
type JobDetail struct {
	*Job
	Registered    bool       `json:"registered"`
	EntryID       int        `json:"entryId,omitempty"`
	RuntimeStatus string     `json:"runtimeStatus"` // scheduled | disabled | unregistered
	NextRun       *time.Time `json:"nextRun,omitempty"`
	LastRun       *time.Time `json:"lastRun,omitempty"`
}

// JobLog 任务执行日志
type JobLog struct {
	RunID     string    `json:"runId"`
	JobID     string    `json:"jobId"`
	JobName   string    `json:"jobName"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Duration  int64     `json:"duration"` // 毫秒
	Success   bool      `json:"success"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
}

// Service 计划任务服务
type Service struct {
	cron    *cronlib.Cron
	store   *Store
	docker  *docker.DockerService      // 可选，DOCKER 类型任务需要
	jobs    map[string]*Job            // jobID → Job
	entries map[string]cronlib.EntryID // jobID → cron entry ID
	mu      sync.RWMutex
}

// AvailableTypes 按当前 OS 及 Docker 可用性返回可用脚本类型
func (s *Service) AvailableTypes() []TypeInfo {
	var types []TypeInfo
	if runtime.GOOS == "windows" {
		types = append(types, []TypeInfo{
			{Value: "BAT", Label: "BAT 批处理脚本"},
			{Value: "POWERSHELL", Label: "PowerShell 脚本"},
			{Value: "EXEC", Label: "可执行文件"},
		}...)
	} else {
		types = append(types, []TypeInfo{
			{Value: "SHELL", Label: "Shell 脚本"},
			{Value: "EXEC", Label: "可执行文件"},
		}...)
	}
	if s.docker != nil {
		types = append(types, []TypeInfo{
			{Value: "DOCKER_TMP", Label: "Docker 临时容器"},
			{Value: "DOCKER_CTR", Label: "Docker 现有容器"},
		}...)
	}
	return types
}

// NewService 创建计划任务服务并启动调度器
func NewService(dockerSvc *docker.DockerService) *Service {
	s := &Service{
		jobs:    make(map[string]*Job),
		entries: make(map[string]cronlib.EntryID),
		cron:    cronlib.New(),
		store:   NewStore(),
		docker:  dockerSvc,
	}

	// 从 cron.yml 加载任务
	jobs, err := s.store.LoadJobs()
	if err != nil {
		logger.Warn("Load cron jobs failed", "error", err)
	}
	for _, job := range jobs {
		if err := s.validateJob(job); err != nil {
			logger.Warn("Skip invalid cron job", "error", err)
			continue
		}
		s.jobs[job.ID] = job
		if job.Enabled {
			if err := s.register(job); err != nil {
				logger.Warn("Cron job register failed", "id", job.ID, "name", job.Name, "error", err)
			}
		}
	}

	s.cron.Start()
	logger.Info("Cron scheduler started", "jobs", len(s.entries))

	// 启动后立即清理一次过期日志，并启动每日清理协程
	s.store.CleanOld()
	cleanCtx, cleanCancel := context.WithCancel(context.Background())
	go s.runLogCleaner(cleanCtx)

	signal.OnQuit(func() {
		cleanCancel()
		ctx := s.cron.Stop()
		<-ctx.Done()
		if err := s.store.Close(); err != nil {
			logger.Warn("Cron log store close failed", "error", err)
		}
		logger.Info("Cron scheduler stopped")
	})

	return s
}

// runLogCleaner 每日凌晨清理过期日志，ctx 取消时退出
func (s *Service) runLogCleaner(ctx context.Context) {
	next := nextCronCleanTime()
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.store.CleanOld()
			next = next.AddDate(0, 0, 1)
			timer.Reset(time.Until(next))
		}
	}
}

// ─── 公开方法 ───

// ListJobs 返回所有任务（含运行状态）
func (s *Service) ListJobs() []*JobDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*JobDetail, 0, len(s.jobs))
	for _, job := range s.jobs {
		detail := &JobDetail{Job: job, RuntimeStatus: "disabled"}
		if job.Enabled {
			detail.RuntimeStatus = "unregistered"
		}
		if entryID, ok := s.entries[job.ID]; ok {
			e := s.cron.Entry(entryID)
			if e.ID == entryID {
				detail.Registered = true
				detail.EntryID = int(entryID)
				detail.RuntimeStatus = "scheduled"
				if !e.Next.IsZero() {
					detail.NextRun = &e.Next
				}
				if !e.Prev.IsZero() {
					detail.LastRun = &e.Prev
				}
			}
		}
		result = append(result, detail)
	}
	return result
}

// CreateJobFromRequest 从请求创建任务（生成 ID、构建 Job、持久化）
func (s *Service) CreateJobFromRequest(req JobUpsertRequest) (*Job, error) {
	job := &Job{
		ID:          strutil.NewString(),
		Name:        req.Name,
		Schedule:    req.Schedule,
		Type:        req.Type,
		Content:     req.Content,
		WorkDir:     req.WorkDir,
		Image:       req.Image,
		Container:   req.Container,
		Volumes:     req.Volumes,
		Timeout:     req.Timeout,
		Enabled:     req.Enabled,
		Description: req.Description,
	}
	if err := s.CreateJob(job); err != nil {
		return nil, err
	}
	return job, nil
}

// UpdateJobFromRequest 从请求更新任务
func (s *Service) UpdateJobFromRequest(id string, req JobUpsertRequest) (*Job, error) {
	if id == "" {
		return nil, fmt.Errorf("任务 ID 不能为空")
	}
	job := &Job{
		ID:          id,
		Name:        req.Name,
		Schedule:    req.Schedule,
		Type:        req.Type,
		Content:     req.Content,
		WorkDir:     req.WorkDir,
		Image:       req.Image,
		Container:   req.Container,
		Volumes:     req.Volumes,
		Timeout:     req.Timeout,
		Enabled:     req.Enabled,
		Description: req.Description,
	}
	if err := s.UpdateJob(job); err != nil {
		return nil, err
	}
	return job, nil
}

// CreateJob 创建任务并持久化
func (s *Service) CreateJob(job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job already exists: %s", job.ID)
	}

	s.jobs[job.ID] = job
	if job.Enabled {
		if err := s.register(job); err != nil {
			delete(s.jobs, job.ID)
			return err
		}
	}

	if err := s.persist(); err != nil {
		if entryID, ok := s.entries[job.ID]; ok {
			s.cron.Remove(entryID)
			delete(s.entries, job.ID)
		}
		delete(s.jobs, job.ID)
		return err
	}
	return nil
}

// UpdateJob 更新任务并重新注册
func (s *Service) UpdateJob(job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldJob, ok := s.jobs[job.ID]
	if !ok {
		return fmt.Errorf("job not found: %s", job.ID)
	}
	oldEntryID, oldEnabled := s.entries[job.ID]

	if oldEnabled {
		s.cron.Remove(oldEntryID)
		delete(s.entries, job.ID)
	}
	s.jobs[job.ID] = job

	if job.Enabled {
		if err := s.register(job); err != nil {
			s.jobs[job.ID] = oldJob
			if oldEnabled {
				if oldEntryID, err := s.cron.AddFunc(oldJob.Schedule, func() { s.runJob(oldJob.ID) }); err == nil {
					s.entries[oldJob.ID] = oldEntryID
				}
			}
			return err
		}
	}

	if err := s.persist(); err != nil {
		if entryID, ok := s.entries[job.ID]; ok {
			s.cron.Remove(entryID)
			delete(s.entries, job.ID)
		}
		s.jobs[job.ID] = oldJob
		if oldEnabled {
			if entryID, err := s.cron.AddFunc(oldJob.Schedule, func() { s.runJob(oldJob.ID) }); err == nil {
				s.entries[oldJob.ID] = entryID
			}
		}
		return err
	}
	return nil
}

// DeleteJob 删除任务
func (s *Service) DeleteJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	oldEntryID, oldEnabled := s.entries[id]
	if oldEnabled {
		s.cron.Remove(oldEntryID)
		delete(s.entries, id)
	}
	delete(s.jobs, id)

	if err := s.persist(); err != nil {
		s.jobs[id] = job
		if oldEnabled {
			if entryID, addErr := s.cron.AddFunc(job.Schedule, func() { s.runJob(job.ID) }); addErr == nil {
				s.entries[id] = entryID
			}
		}
		return err
	}
	return nil
}

// JobStatusPatch 启用或禁用任务
func (s *Service) JobStatusPatch(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Enabled == enabled {
		return nil
	}

	oldEnabled := job.Enabled
	job.Enabled = enabled

	if enabled {
		if err := s.register(job); err != nil {
			job.Enabled = oldEnabled
			return err
		}
	} else {
		if entryID, ok := s.entries[id]; ok {
			s.cron.Remove(entryID)
			delete(s.entries, id)
		}
	}

	if err := s.persist(); err != nil {
		job.Enabled = oldEnabled
		if enabled {
			if entryID, ok := s.entries[id]; ok {
				s.cron.Remove(entryID)
				delete(s.entries, id)
			}
		} else if oldEnabled {
			if entryID, err := s.cron.AddFunc(job.Schedule, func() { s.runJob(job.ID) }); err == nil {
				s.entries[id] = entryID
			}
		}
		return err
	}
	return nil
}

// JobRun 立即触发一次任务（异步执行）
func (s *Service) JobRun(id string) error {
	s.mu.RLock()
	_, ok := s.jobs[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	go s.runJob(id)
	return nil
}

// JobLogs 返回指定任务的执行历史（最近 limit 条，倒序）
// limit <= 0 时默认 50，超过 100 时截断为 100
func (s *Service) JobLogs(id string, limit int) []*JobLog {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.LoadJobLogs(id, limit)
}

// ─── 内部方法 ───

// register 向调度器注册一个任务（调用前须持有锁或在初始化阶段）
func (s *Service) register(job *Job) error {
	entryID, err := s.cron.AddFunc(job.Schedule, func() { s.runJob(job.ID) })
	if err != nil {
		return fmt.Errorf("invalid schedule %q: %w", job.Schedule, err)
	}
	s.entries[job.ID] = entryID
	return nil
}

// persist 将当前 jobs 持久化到 cron.yml（调用前须持有锁）
func (s *Service) persist() error {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return s.store.SaveJobs(jobs)
}

// runJob 执行指定 ID 的任务
func (s *Service) runJob(id string) {
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return
	}

	start := time.Now()
	logger.Info("Cron job running", "id", job.ID, "name", job.Name)

	var output string
	var err error

	if job.Type == "DOCKER_TMP" || job.Type == "DOCKER_CTR" {
		output, err = s.runDockerJob(job)
	} else {
		output, err = command.RunScript(&command.ScriptPayload{
			Name:       job.Name,
			ScriptType: job.Type,
			Content:    job.Content,
			WorkDir:    job.WorkDir,
			Timeout:    job.Timeout,
		})
	}

	end := time.Now()

	entry := &JobLog{
		RunID:     strutil.NewString(),
		JobID:     job.ID,
		JobName:   job.Name,
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start).Milliseconds(),
		Success:   err == nil,
		Output:    output,
	}
	if err != nil {
		entry.Error = err.Error()
		logger.Warn("Cron job failed", "id", job.ID, "name", job.Name, "error", err)
	} else {
		logger.Info("Cron job done", "id", job.ID, "name", job.Name, "duration", entry.Duration)
	}

	s.store.AppendJobLog(entry)
}

// runDockerJob 执行 DOCKER_TMP / DOCKER_CTR 类型任务
func (s *Service) runDockerJob(job *Job) (string, error) {
	if s.docker == nil {
		return "", fmt.Errorf("Docker 服务未启用，无法执行该类型任务")
	}
	switch job.Type {
	case "DOCKER_TMP":
		vols := parseVolumeLines(job.Volumes)
		return s.docker.ContainerRunScript(context.Background(), job.Image, "/bin/sh", job.Content, job.Timeout, vols)
	case "DOCKER_CTR":
		return s.docker.ContainerExecRun(context.Background(), job.Container, "/bin/sh", job.Content, job.Timeout)
	}
	return "", fmt.Errorf("未知的 Docker 任务类型: %s", job.Type)
}

// parseVolumeLines 将换行分隔的 /host:/container[:ro] 字符串转为 VolumeMapping 列表
func parseVolumeLines(s string) []docker.VolumeMapping {
	var result []docker.VolumeMapping
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			continue
		}
		vol := docker.VolumeMapping{
			Type:          "bind",
			Source:        parts[0],
			ContainerPath: parts[1],
		}
		if len(parts) == 3 && strings.Contains(parts[2], "ro") {
			vol.ReadOnly = true
		}
		result = append(result, vol)
	}
	return result
}

func (s *Service) validateJob(job *Job) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}
	if job.ID == "" {
		return fmt.Errorf("job id is required")
	}
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.Schedule == "" {
		return fmt.Errorf("job schedule is required")
	}
	if _, err := cronlib.ParseStandard(job.Schedule); err != nil {
		return fmt.Errorf("invalid schedule %q: %w", job.Schedule, err)
	}
	typeAllowed := false
	for _, item := range s.AvailableTypes() {
		if item.Value == job.Type {
			typeAllowed = true
			break
		}
	}
	if !typeAllowed {
		return fmt.Errorf("unsupported script type on %s: %s", runtime.GOOS, job.Type)
	}
	if job.Content == "" {
		return fmt.Errorf("job content is required")
	}
	if job.Type == "DOCKER_TMP" && job.Image == "" {
		return fmt.Errorf("DOCKER_TMP 类型任务必须指定镜像名")
	}
	if job.Type == "DOCKER_CTR" && job.Container == "" {
		return fmt.Errorf("DOCKER_CTR 类型任务必须指定目标容器名")
	}
	return nil
}
