package monitor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/rehiy/libgo/jsonl"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/docker"
	"isrvd/server/config"
	"isrvd/server/service/notify"
)

// Record 通用监控记录（一行 NDJSON）
// Data 为原始 JSON，存储时不感知具体数据结构
type Record struct {
	Ts          int64           `json:"ts"`                     // 采集时间戳（Unix 秒）
	Data        json.RawMessage `json:"data"`                   // 原始监控数据（主机或容器）
	ContainerID string          `json:"container_id,omitempty"` // 仅容器监控有值
}

// Collector 后台监控采集器，负责定时采集和文件存储
type Collector struct {
	dataDir string
	docker  *docker.DockerService // 保留初始化时的实例，实时查询不读取重载中的全局指针
	host    *jsonl.Store          // 主机监控数据
	ctr     *jsonl.Store          // 容器监控数据
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewCollector 创建采集器。dockerRaw 由调用方注入，为 nil 时跳过容器数据采集。
func NewCollector(dockerRaw *docker.DockerService) *Collector {
	dataDir := filepath.Join(config.Current().Server.RootDirectory, "monitor")
	return &Collector{
		dataDir: dataDir,
		docker:  dockerRaw,
		host:    openStore(dataDir, HostPrefix),
		ctr:     openStore(dataDir, ContainerPrefix),
	}
}

// Start 启动后台采集协程
// 若 config.Current().Monitor.Interval 不合法（非 5/15/30/60）则不启动采集
func (c *Collector) Start(ctx context.Context) {
	if c.cancel != nil {
		return
	}
	interval := time.Duration(config.Current().Monitor.Interval) * time.Second
	if interval <= 0 {
		return
	}

	ctx, c.cancel = context.WithCancel(ctx)
	c.done = make(chan struct{})

	go func() {
		defer close(c.done)
		// 启动后立即采集一次并清理旧文件
		c.collect(ctx)
		CleanOldFiles(c.dataDir)

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
				CleanOldFiles(c.dataDir)
				nextClean = nextClean.AddDate(0, 0, 1)
				cleanTimer.Reset(time.Until(nextClean))
			}
		}
	}()
}

// Stop 停止采集协程并刷盘关闭存储
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
		<-c.done
		c.cancel = nil
	}
	closeStore(c.host)
	closeStore(c.ctr)
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
	if c.docker == nil {
		return nil
	}
	stats, _, err := c.docker.ContainerStats(ctx, id)
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
	if ctx.Err() != nil {
		return
	}
	// ── 主机数据 ──
	stat := CollectHostStat(ctx)
	if ctx.Err() != nil {
		return
	}
	if raw, err := json.Marshal(stat); err == nil {
		appendRecord(c.host, &Record{Ts: time.Now().Unix(), Data: raw})
		c.checkAlert(stat)
	}

	// ── 容器数据 ──
	if c.docker == nil {
		return
	}
	containers, err := c.docker.ContainerList(ctx, false)
	if err != nil {
		logman.Warn("monitor: list containers failed", "error", err)
		return
	}
	for _, ct := range containers {
		if record := c.CollectContainerStatNow(ctx, ct.ID); record != nil {
			record.ContainerID = ct.ID
			appendRecord(c.ctr, record)
		}
	}
}

// checkAlert 将采集数据换算为使用率后交给告警规则检查。
func (c *Collector) checkAlert(stat *HostStat) {
	if stat == nil || stat.System == nil || stat.System.MemoryTotal == 0 {
		return
	}
	sys := stat.System
	cpu := 0.0
	for _, value := range sys.CpuPercent {
		cpu += value
	}
	if len(sys.CpuPercent) > 0 {
		cpu /= float64(len(sys.CpuPercent))
	}
	notify.CheckHost(&notify.HostUsage{
		CPUPercent:    cpu,
		MemoryPercent: float64(sys.MemoryUsed) / float64(sys.MemoryTotal) * 100,
		DiskPercent:   float64(sys.DiskUsed) / float64(sys.DiskTotal) * 100,
	})
}

// History 查询最近 sinceSeconds 秒的历史监控记录；containerID 为空时查询主机
func (c *Collector) History(containerID string, sinceSeconds int64) ([]Record, error) {
	if containerID == "" {
		return readSince(c.host, "", sinceSeconds)
	}
	return readSince(c.ctr, containerID, sinceSeconds)
}

// ─── 辅助函数 ───

// nextMidnight 返回下一个凌晨 00:05 的时间（留 5 分钟余量避免边界问题）
func nextMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
}
