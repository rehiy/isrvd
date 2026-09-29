// Package cron 计划任务业务服务
package cron

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/strutil"
	"github.com/robfig/cron/v3"

	"isrvd/pkgs/docker"
	"isrvd/server/config"
	"isrvd/server/service/notify"
)

var (
	logger        = logman.Named("cron")
	ErrJobRunning = errors.New("任务正在运行")
	activeJobs    sync.Map
)

// TypeInfo 脚本类型描述
type TypeInfo struct {
	Value string `json:"value"` // 类型值：SHELL | EXEC | BAT | POWERSHELL | DOCKER_TMP | DOCKER_CTR
	Label string `json:"label"` // 类型显示名称
}

// Job 计划任务业务类型
type Job struct {
	ID          string `yaml:"id" json:"id"`                                   // 任务 ID（自动生成）
	Name        string `yaml:"name" json:"name"`                               // 任务名称
	Schedule    string `yaml:"schedule" json:"schedule"`                       // cron 表达式（如 "0 3 * * *"）
	Type        string `yaml:"type" json:"type"`                               // 脚本类型：SHELL | EXEC | BAT | POWERSHELL | DOCKER_TMP | DOCKER_CTR
	Content     string `yaml:"content" json:"content"`                         // 脚本内容或命令
	WorkDir     string `yaml:"workDir" json:"workDir"`                         // 工作目录（SHELL/EXEC 类型）
	Image       string `yaml:"image,omitempty" json:"image,omitempty"`         // DOCKER_TMP：镜像名
	Container   string `yaml:"container,omitempty" json:"container,omitempty"` // DOCKER_CTR：目标容器名
	Volumes     string `yaml:"volumes,omitempty" json:"volumes,omitempty"`     // DOCKER_TMP：额外挂载，格式：/host:/container[:ro]，换行分隔
	Timeout     uint   `yaml:"timeout" json:"timeout"`                         // 超时（秒），0 表示不限制
	Enabled     bool   `yaml:"enabled" json:"enabled"`                         // 是否启用
	Description string `yaml:"description" json:"description"`                 // 任务描述
}

// Service 计划任务服务
type Service struct {
	cron          *cron.Cron              // cron 调度器实例
	store         *Store                  // 任务持久化存储
	docker        *docker.DockerService   // 可选，DOCKER 类型任务需要
	jobs          map[string]*Job         // jobID → Job 映射
	entries       map[string]cron.EntryID // jobID → cron entry ID 映射
	ctx           context.Context
	cancel        context.CancelFunc
	running       map[string]context.CancelFunc
	closeOnce     sync.Once
	closeDone     chan struct{}
	workers       sync.WaitGroup // 任务执行与日志清理共用等待组
	closed        bool
	failureNotify func(string, string, string, int64)
	mu            sync.RWMutex // 保护任务、调度条目和关闭状态
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

// NewService 创建计划任务服务并启动调度器。dockerRaw 由调用方注入，为 nil 时禁用 DOCKER 类型任务。
func NewService(parent context.Context, dockerRaw *docker.DockerService) *Service {
	if parent == nil {
		parent = context.Background()
	}
	cleanCtx, cleanCancel := context.WithCancel(parent)
	snapshot := config.Current()
	s := &Service{
		jobs:          make(map[string]*Job),
		entries:       make(map[string]cron.EntryID),
		cron:          cron.New(),
		store:         NewStore(snapshot.Server.RootDirectory),
		docker:        dockerRaw,
		ctx:           parent,
		cancel:        cleanCancel,
		running:       make(map[string]context.CancelFunc),
		closeDone:     make(chan struct{}),
		failureNotify: notify.JobFailureNotifier(snapshot.Notify),
	}

	// 从 cron.yml 加载任务
	jobs, err := s.store.LoadJobs()
	if err != nil {
		logger.Warn("Load cron jobs failed", "error", err)
	}
	migrated := false
	for _, job := range jobs {
		if job.Type == "DOCKER" {
			switch {
			case job.Container != "":
				job.Type = "DOCKER_CTR"
				migrated = true
			case job.Image != "":
				job.Type = "DOCKER_TMP"
				migrated = true
			}
		}
		if err := s.validateJob(job); err != nil {
			logger.Warn("Skip invalid cron job", "error", err)
			continue
		}
		if _, exists := s.jobs[job.ID]; exists {
			logger.Warn("Skip duplicate cron job", "id", job.ID)
			continue
		}
		s.jobs[job.ID] = job
		if job.Enabled {
			if err := s.register(job); err != nil {
				logger.Warn("Cron job register failed", "id", job.ID, "name", job.Name, "error", err)
			}
		}
	}
	if migrated {
		if err := s.store.SaveJobs(jobs); err != nil {
			logger.Warn("Cron legacy job migration save failed", "error", err)
		}
	}

	s.cron.Start()
	logger.Info("Cron scheduler started", "jobs", len(s.entries))

	// 启动后立即清理一次过期日志，并启动每日清理协程
	s.store.CleanOld()
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		s.runLogCleaner(cleanCtx)
	}()

	return s
}

// Close 停止调度和清理协程；已接受的任务继续执行，完成后关闭日志存储。
// 父 context 取消时，仍在执行的任务会被取消。
func (s *Service) Close() <-chan struct{} {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.cancel()
		s.cron.Stop()
		s.mu.Unlock()

		go func() {
			s.workers.Wait()
			if err := s.store.Close(); err != nil {
				logger.Warn("Cron log store close failed", "error", err)
			}
			logger.Info("Cron scheduler stopped")
			close(s.closeDone)
		}()
	})
	return s.closeDone
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

// JobDetail 任务详情（含运行时调度状态）
type JobDetail struct {
	*Job
	Registered    bool       `json:"registered"`        // 是否已注册到调度器
	EntryID       int        `json:"entryId,omitempty"` // cron 条目 ID
	RuntimeStatus string     `json:"runtimeStatus"`     // 运行状态：scheduled | disabled | unregistered
	NextRun       *time.Time `json:"nextRun,omitempty"` // 下次运行时间
	LastRun       *time.Time `json:"lastRun,omitempty"` // 上次运行时间
}

// JobList 返回所有任务（含运行状态）
func (s *Service) JobList() []*JobDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*JobDetail, 0, len(s.jobs))
	for _, job := range s.jobs {
		detail := &JobDetail{Job: cloneJob(job), RuntimeStatus: "disabled"}
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

// JobUpsertRequest 创建/更新任务请求（由 server 层传入，service 层负责构建 Job）
type JobUpsertRequest struct {
	Name        string `json:"name" binding:"required"`     // 任务名称
	Schedule    string `json:"schedule" binding:"required"` // cron 表达式
	Type        string `json:"type" binding:"required"`     // 脚本类型
	Content     string `json:"content" binding:"required"`  // 脚本内容
	WorkDir     string `json:"workDir"`                     // 工作目录
	Image       string `json:"image"`                       // DOCKER_TMP：镜像名
	Container   string `json:"container"`                   // DOCKER_CTR：目标容器名
	Volumes     string `json:"volumes"`                     // DOCKER_TMP：额外挂载
	Timeout     uint   `json:"timeout"`                     // 超时（秒）
	Enabled     bool   `json:"enabled"`                     // 是否启用
	Description string `json:"description"`                 // 任务描述
}

// JobCreateFromRequest 从请求创建任务（生成 ID、构建 Job、持久化）
func (s *Service) JobCreateFromRequest(req JobUpsertRequest) (*Job, error) {
	job := s.jobFromRequest(strutil.NewString(), req)
	if err := s.JobCreate(job); err != nil {
		return nil, err
	}
	return job, nil
}

// JobUpdateFromRequest 从请求更新任务
func (s *Service) JobUpdateFromRequest(id string, req JobUpsertRequest) (*Job, error) {
	if id == "" {
		return nil, fmt.Errorf("任务 ID 不能为空")
	}
	job := s.jobFromRequest(id, req)
	if err := s.JobUpdate(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Service) jobFromRequest(id string, req JobUpsertRequest) *Job {
	workDir := config.PathToAbs(req.WorkDir, s.store.rootDir)
	return &Job{
		ID:          id,
		Name:        req.Name,
		Schedule:    req.Schedule,
		Type:        req.Type,
		Content:     req.Content,
		WorkDir:     workDir,
		Image:       req.Image,
		Container:   req.Container,
		Volumes:     req.Volumes,
		Timeout:     req.Timeout,
		Enabled:     req.Enabled,
		Description: req.Description,
	}
}

// JobCreate 创建任务并持久化
func (s *Service) JobCreate(job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}
	storedJob := cloneJob(job)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("cron scheduler stopped")
	}

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job already exists: %s", job.ID)
	}

	return s.jobReplace(storedJob.ID, storedJob)
}

// JobUpdate 更新任务并重新注册
func (s *Service) JobUpdate(job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}
	storedJob := cloneJob(job)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("cron scheduler stopped")
	}

	if _, ok := s.jobs[job.ID]; !ok {
		return fmt.Errorf("job not found: %s", job.ID)
	}
	return s.jobReplace(storedJob.ID, storedJob)
}

// JobDelete 删除任务
func (s *Service) JobDelete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("cron scheduler stopped")
	}

	if _, ok := s.jobs[id]; !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	return s.jobReplace(id, nil)
}

// JobStatusPatch 启用或禁用任务
func (s *Service) JobStatusPatch(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("cron scheduler stopped")
	}

	oldJob, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if oldJob.Enabled == enabled {
		return nil
	}

	job := cloneJob(oldJob)
	job.Enabled = enabled
	return s.jobReplace(id, job)
}

// JobRun 立即触发一次任务（异步执行）
func (s *Service) JobRun(id string) error {
	job, ctx, err := s.prepareRun(id)
	if err != nil {
		return err
	}
	go s.executeJob(ctx, job)
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

// jobReplace 同步任务、调度条目与持久化；失败时恢复原状态。nil 表示删除。
// 调用前须持有写锁。
func (s *Service) jobReplace(id string, job *Job) error {
	oldJob := s.jobs[id]
	_, registered := s.entries[id]
	s.unregister(id)
	var err error
	if job == nil {
		delete(s.jobs, id)
	} else {
		s.jobs[id] = job
		if job.Enabled {
			err = s.register(job)
		}
	}
	if err == nil {
		err = s.persist()
	}
	if err == nil {
		return nil
	}

	s.unregister(id)
	if oldJob == nil {
		delete(s.jobs, id)
	} else {
		s.jobs[id] = oldJob
		if registered {
			if restoreErr := s.register(oldJob); restoreErr != nil {
				logger.Warn("Cron job restore failed", "id", id, "error", restoreErr)
			}
		}
	}
	return err
}

// register 向调度器注册一个任务（调用前须持有锁或在初始化阶段）
func (s *Service) register(job *Job) error {
	entryID, err := s.cron.AddFunc(job.Schedule, func() { s.runJob(job.ID) })
	if err != nil {
		return fmt.Errorf("invalid schedule %q: %w", job.Schedule, err)
	}
	s.entries[job.ID] = entryID
	return nil
}

// unregister 移除任务的调度条目（调用前须持有锁）。
func (s *Service) unregister(id string) {
	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}

// persist 将当前 jobs 持久化到 cron.yml（调用前须持有锁）
func (s *Service) persist() error {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return s.store.SaveJobs(jobs)
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copy := *job
	return &copy
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
	if _, err := cron.ParseStandard(job.Schedule); err != nil {
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

// ─── 辅助函数 ───

// nextCronCleanTime 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextCronCleanTime() time.Time {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 5, 0, 0, tomorrow.Location())
}
