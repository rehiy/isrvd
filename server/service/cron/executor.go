package cron

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types/mount"
	"github.com/rehiy/libgo/command"
	"github.com/rehiy/libgo/strutil"
)

type jobOutputBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (b *jobOutputBuffer) Write(p []byte) (int, error) {
	remaining := maxStoredJobOutputBytes - b.buf.Len()
	if remaining > 0 {
		keep := min(len(p), remaining)
		_, _ = b.buf.Write(p[:keep])
	}
	if len(p) > remaining {
		b.truncated = true
	}
	return len(p), nil
}

func (b *jobOutputBuffer) String() string {
	value := b.buf.String()
	if b.truncated {
		value += jobOutputTruncatedMark
	}
	return truncateJobOutput(value)
}

// runJob 执行指定 ID 的任务
func (s *Service) runJob(id string) {
	job, ctx, err := s.prepareRun(id)
	if errors.Is(err, ErrJobRunning) {
		logger.Info("Cron job skipped because previous run is active", "id", id)
		return
	}
	if err != nil {
		return
	}
	s.executeJob(ctx, job)
}

// prepareRun 在同一把锁内接受任务、建立取消上下文并阻止同一任务重入。
func (s *Service) prepareRun(id string) (*Job, context.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, nil, fmt.Errorf("cron scheduler stopped")
	}
	job, ok := s.jobs[id]
	if !ok {
		return nil, nil, fmt.Errorf("job not found: %s", id)
	}
	if _, running := s.running[id]; running {
		return nil, nil, ErrJobRunning
	}
	if _, running := activeJobs.LoadOrStore(id, struct{}{}); running {
		return nil, nil, ErrJobRunning
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.running[id] = cancel
	s.workers.Add(1)
	return cloneJob(job), ctx, nil
}

func (s *Service) finishRun(id string) {
	s.mu.Lock()
	if cancel, ok := s.running[id]; ok {
		cancel()
		delete(s.running, id)
	}
	s.mu.Unlock()
	activeJobs.Delete(id)
	s.workers.Done()
}

func (s *Service) executeJob(ctx context.Context, job *Job) {
	defer s.finishRun(job.ID)

	start := time.Now()
	logger.Info("Cron job running", "id", job.ID, "name", job.Name)

	var output string
	var err error

	if job.Type == "DOCKER_TMP" || job.Type == "DOCKER_CTR" {
		output, err = s.runDockerJob(ctx, job)
	} else {
		output, err = runLocalJob(ctx, job)
	}

	end := time.Now()
	serviceCanceled := errors.Is(err, context.Canceled) && s.ctx.Err() != nil

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
	if err != nil && !serviceCanceled {
		s.failureNotify(entry.JobID, entry.JobName, entry.RunID, entry.Duration)
	}
}

// runDockerJob 执行 DOCKER_TMP / DOCKER_CTR 类型任务
func (s *Service) runDockerJob(ctx context.Context, job *Job) (string, error) {
	if s.docker == nil {
		return "", fmt.Errorf("Docker 服务未启用，无法执行该类型任务")
	}
	switch job.Type {
	case "DOCKER_TMP":
		vols := parseVolumeLines(job.Volumes)
		return s.docker.ContainerRunScript(ctx, job.Image, "/bin/sh", job.Content, job.Timeout, vols)
	case "DOCKER_CTR":
		return s.docker.ContainerExecRun(ctx, job.Container, "/bin/sh", job.Content, job.Timeout)
	}
	return "", fmt.Errorf("未知的 Docker 任务类型: %s", job.Type)
}

// ─── 辅助函数 ───

func normalizeScriptContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	if runtime.GOOS == "windows" {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	return content
}

func runLocalJob(parent context.Context, job *Job) (string, error) {
	ctx := parent
	var cancel context.CancelFunc
	if job.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, time.Duration(job.Timeout)*time.Second)
		defer cancel()
	}

	var bin string
	var args []string
	var tempPath string
	switch job.Type {
	case "BAT":
		file, err := os.CreateTemp("", "isrvd-cron-*.bat")
		if err != nil {
			return "", err
		}
		tempPath = file.Name()
		if _, err := file.WriteString(normalizeScriptContent(job.Content)); err != nil {
			file.Close()
			os.Remove(tempPath)
			return "", err
		}
		if err := file.Close(); err != nil {
			os.Remove(tempPath)
			return "", err
		}
		defer os.Remove(tempPath)
		bin, args = command.GetShell("cmd"), []string{"/c", "CALL", tempPath}
	case "POWERSHELL":
		file, err := os.CreateTemp("", "isrvd-cron-*.ps1")
		if err != nil {
			return "", err
		}
		tempPath = file.Name()
		if _, err := file.WriteString(normalizeScriptContent(job.Content)); err != nil {
			file.Close()
			os.Remove(tempPath)
			return "", err
		}
		if err := file.Close(); err != nil {
			os.Remove(tempPath)
			return "", err
		}
		defer os.Remove(tempPath)
		bin, args = command.GetShell("powershell"), []string{"-File", tempPath}
	case "SHELL":
		if strings.HasPrefix(job.Content, "#!/") {
			file, err := os.CreateTemp("", "isrvd-cron-*")
			if err != nil {
				return "", err
			}
			tempPath = file.Name()
			if _, err := file.WriteString(normalizeScriptContent(job.Content)); err != nil {
				file.Close()
				os.Remove(tempPath)
				return "", err
			}
			if err := file.Close(); err != nil {
				os.Remove(tempPath)
				return "", err
			}
			if err := os.Chmod(tempPath, 0700); err != nil {
				os.Remove(tempPath)
				return "", err
			}
			defer os.Remove(tempPath)
			bin = tempPath
		} else {
			bin, args = command.DefaultShell(), []string{"-c", job.Content}
		}
	case "EXEC":
		fields := strings.Fields(job.Content)
		if len(fields) == 0 {
			return "", fmt.Errorf("执行命令不能为空")
		}
		bin, args = fields[0], fields[1:]
	default:
		return "", fmt.Errorf("unsupported script type: %s", job.Type)
	}

	workDir := job.WorkDir
	if workDir == "" {
		workDir = filepath.Dir(bin)
	}
	cmd := command.NewCommand(ctx, bin, args, workDir)
	cmd.WaitDelay = 5 * time.Second
	output := &jobOutputBuffer{}
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	if ctx.Err() != nil {
		return output.String(), ctx.Err()
	}
	return output.String(), err
}

// parseVolumeLines 将换行分隔的 /host:/container[:ro] 字符串转为 Docker mount 列表
func parseVolumeLines(s string) []mount.Mount {
	var result []mount.Mount
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			continue
		}
		vol := mount.Mount{
			Type:   mount.TypeBind,
			Source: parts[0],
			Target: parts[1],
		}
		if len(parts) == 3 && strings.Contains(parts[2], "ro") {
			vol.ReadOnly = true
		}
		result = append(result, vol)
	}
	return result
}
