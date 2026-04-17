package system

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/psutil"
	"github.com/shirou/gopsutil/v3/disk"

	"isrvd/config"
	"isrvd/internal/helper"
	"isrvd/internal/registry"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// DiskIOStat 硬盘 IO 统计
type DiskIOStat struct {
	Name       string `json:"Name"`
	ReadBytes  uint64 `json:"ReadBytes"`
	WriteBytes uint64 `json:"WriteBytes"`
	ReadCount  uint64 `json:"ReadCount"`
	WriteCount uint64 `json:"WriteCount"`
}

// SystemStatResponse 系统统计响应
type SystemStatResponse struct {
	System *psutil.DetailStat `json:"system"`
	DiskIO []*DiskIOStat      `json:"diskIO"`
	Go     *GoRuntimeStat     `json:"go"`
}

// GoRuntimeStat Go 运行态统计
type GoRuntimeStat struct {
	Version      string `json:"version"`
	NumCPU       int    `json:"numCPU"`
	NumGoroutine int    `json:"numGoroutine"`
	*psutil.GoMemoryStat
}

// filterDiskPartitions 过滤磁盘分区，去除容器中的重复单文件挂载
// 同一设备+相同容量的分区只保留挂载点路径最短的那个
func filterDiskPartitions(partitions []psutil.DiskPartition) []psutil.DiskPartition {
	type bestEntry struct {
		index int
		mpLen int
	}
	best := make(map[string]*bestEntry)

	for i, dp := range partitions {
		key := fmt.Sprintf("%s:%d", dp.Device, dp.Total)
		if b, ok := best[key]; !ok || len(dp.Mountpoint) < b.mpLen {
			best[key] = &bestEntry{index: i, mpLen: len(dp.Mountpoint)}
		}
	}

	result := make([]psutil.DiskPartition, 0, len(best))
	for _, b := range best {
		result = append(result, partitions[b.index])
	}
	return result
}

// Stat 获取系统统计信息
func (h *SystemHandler) Stat(c *gin.Context) {
	detail := psutil.Detail(false)

	// 过滤容器中的单文件挂载项，按 "设备+容量" 去重，只保留挂载点最短的
	detail.DiskPartition = filterDiskPartitions(detail.DiskPartition)

	// 采集硬盘 IO 数据
	ioCounters, _ := disk.IOCounters()
	diskIO := make([]*DiskIOStat, 0, len(ioCounters))
	for name, counter := range ioCounters {
		diskIO = append(diskIO, &DiskIOStat{
			Name:       name,
			ReadBytes:  counter.ReadBytes,
			WriteBytes: counter.WriteBytes,
			ReadCount:  counter.ReadCount,
			WriteCount: counter.WriteCount,
		})
	}

	goStat := &GoRuntimeStat{
		Version:      runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		GoMemoryStat: psutil.GoMemory(),
	}

	helper.RespondSuccess(c, "ok", &SystemStatResponse{
		System: detail,
		DiskIO: diskIO,
		Go:     goStat,
	})
}

// Probe 探活
func (h *SystemHandler) Probe(c *gin.Context) {
	ctx := c.Request.Context()
	helper.RespondSuccess(c, "ok", gin.H{
		"agent":  gin.H{"available": config.Agent.BaseURL != "" && config.Agent.APIKey != ""},
		"docker": gin.H{"available": registry.IsDockerAvailable(ctx)},
		"swarm":  gin.H{"available": registry.IsSwarmAvailable(ctx)},
		"apisix": gin.H{"available": registry.IsApisixAvailable()},
	})
}

// UpTime 获取服务启动时间（秒级时间戳）
var startTime = time.Now()

func (h *SystemHandler) Uptime(c *gin.Context) {
	helper.RespondSuccess(c, "ok", gin.H{
		"startTime": startTime.Unix(),
		"uptime":    int64(time.Since(startTime).Seconds()),
	})
}

// Health 健康检查
func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
