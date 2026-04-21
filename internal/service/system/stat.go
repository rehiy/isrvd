// Package system 提供系统信息业务服务层
package system

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/rehiy/pango/psutil"
	"github.com/shirou/gopsutil/v3/disk"

	"isrvd/config"
	"isrvd/internal/registry"
)

// DiskIOStat 硬盘 IO 统计
type DiskIOStat struct {
	Name       string `json:"Name"`
	ReadBytes  uint64 `json:"ReadBytes"`
	WriteBytes uint64 `json:"WriteBytes"`
	ReadCount  uint64 `json:"ReadCount"`
	WriteCount uint64 `json:"WriteCount"`
}

// GoRuntimeStat Go 运行态统计
type GoRuntimeStat struct {
	Version      string `json:"version"`
	NumCPU       int    `json:"numCPU"`
	NumGoroutine int    `json:"numGoroutine"`
	*psutil.GoMemoryStat
}

// SystemStatResponse 系统统计响应
type SystemStatResponse struct {
	System *psutil.DetailStat `json:"system"`
	DiskIO []*DiskIOStat      `json:"diskIO"`
	Go     *GoRuntimeStat     `json:"go"`
}

// ProbeResponse 探活响应
type ProbeResponse struct {
	Agent  map[string]bool `json:"agent"`
	Docker map[string]bool `json:"docker"`
	Swarm  map[string]bool `json:"swarm"`
	Apisix map[string]bool `json:"apisix"`
}

// UptimeResponse 服务启动时间响应
type UptimeResponse struct {
	StartTime int64 `json:"startTime"`
	Uptime    int64 `json:"uptime"`
}

var startTime = time.Now()

// Service 系统信息业务服务
type Service struct{}

// NewService 创建系统信息业务服务
func NewService() *Service {
	return &Service{}
}

// filterDiskPartitions 过滤磁盘分区，去除容器中的重复单文件挂载
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
func (s *Service) Stat() *SystemStatResponse {
	detail := psutil.Detail(false)
	detail.DiskPartition = filterDiskPartitions(detail.DiskPartition)

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

	return &SystemStatResponse{System: detail, DiskIO: diskIO, Go: goStat}
}

// Probe 探活
func (s *Service) Probe(ctx context.Context) *ProbeResponse {
	return &ProbeResponse{
		Agent:  map[string]bool{"available": config.Agent.BaseURL != "" && config.Agent.APIKey != ""},
		Docker: map[string]bool{"available": registry.IsDockerAvailable(ctx)},
		Swarm:  map[string]bool{"available": registry.IsSwarmAvailable(ctx)},
		Apisix: map[string]bool{"available": registry.IsApisixAvailable()},
	}
}

// Uptime 获取服务启动时间
func (s *Service) Uptime() *UptimeResponse {
	return &UptimeResponse{
		StartTime: startTime.Unix(),
		Uptime:    int64(time.Since(startTime).Seconds()),
	}
}
