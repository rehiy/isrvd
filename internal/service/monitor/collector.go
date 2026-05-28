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

const (
	// collectInterval 采集间隔
	collectInterval = 15 * time.Second

	// hostPrefix 主机监控文件前缀
	hostPrefix = "host"
	// containerPrefix 容器监控文件前缀
	containerPrefix = "ctr"
)

// Record 通用监控记录（一行 NDJSON）
// Data 为原始 JSON，存储时不感知具体数据结构
type Record struct {
	Ts   int64           `json:"ts"`
	Data json.RawMessage `json:"data"`
}

// Collector 后台监控采集器，负责定时采集和文件存储
type Collector struct {
	dataDir string
}

// NewCollector 创建采集器
func NewCollector() *Collector {
	return &Collector{
		dataDir: filepath.Join(config.Server.RootDirectory, "monitor"),
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

// collect 执行一次采集
func (c *Collector) collect(ctx context.Context) {
	ts := time.Now().Unix()

	// ── 主机数据 ──
	c.Append(hostPrefix, ts, CollectHostStat(ctx))

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
		stats, err := registry.DockerService.ContainerStats(ctx, ct.ID)
		if err != nil {
			continue
		}
		c.Append(containerPrefix+"_"+ct.ID, ts, stats)
	}
}

// Append 将任意数据序列化后追加到指定前缀的当天文件
func (c *Collector) Append(prefix string, ts int64, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		logman.Warn("monitor: marshal data failed", "prefix", prefix, "error", err)
		return
	}
	if err := AppendRecord(c.dataDir, prefix, &Record{Ts: ts, Data: raw}); err != nil {
		logman.Warn("monitor: write record failed", "prefix", prefix, "error", err)
	}
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

// cleanOld 清理所有过期文件
func (c *Collector) cleanOld() {
	CleanOldFiles(c.dataDir)
}

// nextMidnight 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
}
