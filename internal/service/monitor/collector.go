package monitor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	"isrvd/internal/registry"
)

// Record 通用监控记录（一行 NDJSON）
// Data 为原始 JSON，存储时不感知具体数据结构
type Record struct {
	Ts          int64           `json:"ts"`
	Data        json.RawMessage `json:"data"`
	ContainerID string          `json:"container_id,omitempty"` // 仅容器监控有值
}

// Collector 后台监控采集器，负责定时采集和文件存储
type Collector struct {
	dataDir string
	cancel  context.CancelFunc
}

// NewCollector 创建采集器
func NewCollector() *Collector {
	return &Collector{
		dataDir: filepath.Join(config.Server.RootDirectory, "monitor"),
	}
}

// Start 启动后台采集协程
// 若 config.Monitor.Interval 不合法（非 5/15/30/60）则不启动采集
func (c *Collector) Start(ctx context.Context) {
	interval := time.Duration(config.Monitor.Interval) * time.Second
	if interval <= 0 {
		return
	}

	ctx, c.cancel = context.WithCancel(ctx)

	go func() {
		// 启动后立即采集一次并清理旧文件
		c.collect(ctx)
		c.cleanOld()

		ticker := time.NewTicker(interval)
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

// Stop 停止采集协程
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

// Restart 重启采集协程（用于 reload 后使新配置生效）
// 同时更新 dataDir，确保 RootDirectory 变更后写入正确目录。
func (c *Collector) Restart(ctx context.Context) {
	c.Stop()
	c.dataDir = filepath.Join(config.Server.RootDirectory, "monitor")
	c.Start(ctx)
}

// CollectHostStatNow 实时采集主机数据，不写入文件
func (c *Collector) CollectHostStatNow(ctx context.Context) *Record {
	data := CollectHostStat(ctx)
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return &Record{Ts: time.Now().Unix(), Data: raw}
}

// CollectContainerStatNow 实时采集指定容器数据，不写入文件
func (c *Collector) CollectContainerStatNow(ctx context.Context, id string) *Record {
	if registry.DockerService == nil {
		return nil
	}
	stats, _, err := registry.DockerService.ContainerStats(ctx, id)
	if err != nil {
		return nil
	}
	raw, err := json.Marshal(stats)
	if err != nil {
		return nil
	}
	return &Record{Ts: time.Now().Unix(), Data: raw}
}

// collect 执行一次采集
func (c *Collector) collect(ctx context.Context) {
	// ── 主机数据 ──
	record := c.CollectHostStatNow(ctx)
	if record != nil {
		AppendRawRecord(c.dataDir, HostPrefix, "", record.Ts, record.Data)
	}

	// ── 容器数据 ──
	if registry.DockerService == nil {
		return
	}
	containers, err := registry.DockerService.ContainerList(ctx, false)
	if err != nil {
		logman.Warn("monitor: list containers failed", "error", err)
		return
	}
	for _, ct := range containers {
		record := c.CollectContainerStatNow(ctx, ct.ID)
		if record != nil {
			AppendRawRecord(c.dataDir, ContainerPrefix, ct.ID, record.Ts, record.Data)
		}
	}
}

// DataDir 返回数据目录
func (c *Collector) DataDir() string {
	return c.dataDir
}

// cleanOld 清理所有过期文件
func (c *Collector) cleanOld() {
	CleanOldFiles(c.dataDir)
}

// nextMidnight 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
}
