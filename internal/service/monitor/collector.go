package monitor

import (
	"context"
	"path/filepath"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	svcOverview "isrvd/internal/service/overview"
	pkgdocker "isrvd/pkgs/docker"
)

const (
	// collectInterval 采集间隔
	collectInterval = 15 * time.Second

	// hostPrefix 主机监控文件前缀
	hostPrefix = "host"
	// containerPrefix 容器监控文件前缀
	containerPrefix = "ctr"
)

// HostRecord 主机监控记录（一行 NDJSON）
type HostRecord struct {
	Ts   int64                     `json:"ts"`
	Data *svcOverview.StatResponse `json:"data"`
}

// ContainerRecord 容器监控记录（一行 NDJSON）
type ContainerRecord struct {
	Ts   int64                             `json:"ts"`
	Data *pkgdocker.ContainerStatsResponse `json:"data"`
}

// Collector 后台监控采集器
type Collector struct {
	overviewSvc *svcOverview.Service
	dockerSvc   *pkgdocker.DockerService // 可为 nil（无 Docker 时跳过容器采集）
	dataDir     string
}

// NewCollector 创建采集器
func NewCollector(overviewSvc *svcOverview.Service, dockerSvc *pkgdocker.DockerService) *Collector {
	dataDir := filepath.Join(config.Server.RootDirectory, "monitor")
	return &Collector{
		overviewSvc: overviewSvc,
		dockerSvc:   dockerSvc,
		dataDir:     dataDir,
	}
}

// Start 启动后台采集协程，ctx 取消时退出
func (c *Collector) Start(ctx context.Context) {
	go func() {
		// 启动后立即采集一次并清理旧文件
		c.collect(ctx)
		c.cleanOld()

		ticker := time.NewTicker(collectInterval)
		defer ticker.Stop()

		// 每日清理定时器：计算到下一个凌晨 00:05 的等待时间
		nextClean := nextMidnight()
		cleanTimer := time.NewTimer(time.Until(nextClean))
		defer cleanTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collect(ctx)
			case <-cleanTimer.C:
				c.cleanOld()
				nextClean = nextClean.AddDate(0, 0, 1)
				cleanTimer.Reset(time.Until(nextClean))
			}
		}
	}()
}

// DataDir 返回数据目录
func (c *Collector) DataDir() string {
	return c.dataDir
}

// HostFilePrefix 返回主机监控文件前缀
func HostFilePrefix() string {
	return hostPrefix
}

// ContainerFilePrefix 返回指定容器监控文件前缀
func ContainerFilePrefix(id string) string {
	return containerPrefix + "_" + id
}

// collect 执行一次采集
func (c *Collector) collect(ctx context.Context) {
	ts := time.Now().Unix()

	// ── 主机数据 ──
	stat := c.overviewSvc.Stat(ctx)
	rec := &HostRecord{Ts: ts, Data: stat}
	if err := AppendRecord(c.dataDir, hostPrefix, rec); err != nil {
		logman.Warn("monitor: write host record failed", "error", err)
	}

	// ── 容器数据 ──
	if c.dockerSvc == nil {
		return
	}

	containers, err := c.dockerSvc.ContainerList(ctx, false) // 只采集运行中的容器
	if err != nil {
		logman.Warn("monitor: list containers failed", "error", err)
		return
	}

	for _, ct := range containers {
		id := ct.ID
		stats, err := c.dockerSvc.ContainerStats(ctx, id)
		if err != nil {
			// 单个容器失败不影响其他容器
			continue
		}
		crec := &ContainerRecord{Ts: ts, Data: stats}
		if err := AppendRecord(c.dataDir, containerPrefix+"_"+id, crec); err != nil {
			logman.Warn("monitor: write container record failed", "id", id, "error", err)
		}
	}
}

// cleanOld 清理所有过期文件
func (c *Collector) cleanOld() {
	CleanOldFiles(c.dataDir)
}

// UpdateDockerSvc 更新 Docker 服务（用于 reload 后同步最新实例）
func (c *Collector) UpdateDockerSvc(dockerSvc *pkgdocker.DockerService) {
	c.dockerSvc = dockerSvc
}

// nextMidnight 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextMidnight() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
	return next
}
