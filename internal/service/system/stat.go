// Package system 提供系统信息业务服务层
package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/rehiy/pango/psutil"
	"github.com/shirou/gopsutil/v3/disk"

	"isrvd/config"
	"isrvd/internal/registry"
	pkggpu "isrvd/pkgs/gpu"
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

// SystemGPU 系统统计接口使用的 GPU 响应结构
// index 仅用于展示排序；deviceKey 用于稳定标识设备
// deviceKey 优先使用 PCI 地址，无法获取时退化为 vendor/name/index 组合
// 这样前端图表与列表 key 不会因过滤顺序变化而漂移
type SystemGPU struct {
	Index       int     `json:"index"`
	DeviceKey   string  `json:"deviceKey"`
	Name        string  `json:"name"`
	Vendor      string  `json:"vendor"`
	MemoryUsed  uint64  `json:"memoryUsed"`
	MemoryTotal uint64  `json:"memoryTotal"`
	Utilization float64 `json:"utilization"`
	Temperature int     `json:"temperature"`
	PowerUsage  float64 `json:"powerUsage"`
	FanSpeed    int     `json:"fanSpeed"`
}

// SystemStatResponse 系统统计响应
type SystemStatResponse struct {
	System       *psutil.DetailStat `json:"system"`
	DiskIO       []*DiskIOStat      `json:"diskIO"`
	GPU          []*SystemGPU       `json:"gpu"`
	Go           *GoRuntimeStat     `json:"go"`
	Version      string             `json:"version"`
	VersionCheck *VersionCheck      `json:"versionCheck,omitempty"`
}

// ProbeResponse 探活响应
type ProbeResponse struct {
	Agent   map[string]bool `json:"agent"`
	Apisix  map[string]bool `json:"apisix"`
	Docker  map[string]bool `json:"docker"`
	Swarm   map[string]bool `json:"swarm"`
	Compose map[string]bool `json:"compose"`
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

func buildGPUDeviceKey(vendor, address, name string, index int) string {
	address = strings.TrimSpace(strings.ToLower(address))
	if address != "" {
		return vendor + ":" + address
	}

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "gpu"
	}
	name = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", ".", "-", "\"", "", "'", "").Replace(name)
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	return fmt.Sprintf("%s:%s:%d", vendor, name, index)
}

func buildSystemGPUs(ctx context.Context) []*SystemGPU {
	deviceStats, err := pkggpu.GetGPUStats(ctx)
	if err != nil && len(deviceStats) == 0 {
		return nil
	}

	systemGPUs := make([]*SystemGPU, 0, len(deviceStats))
	for i, stat := range deviceStats {
		systemGPUs = append(systemGPUs, &SystemGPU{
			Index:       i,
			DeviceKey:   buildGPUDeviceKey(stat.Vendor, stat.Address, stat.Name, i),
			Name:        stat.Name,
			Vendor:      stat.Vendor,
			MemoryUsed:  stat.MemoryUsed,
			MemoryTotal: stat.MemoryTotal,
			Utilization: stat.Utilization,
			Temperature: stat.Temperature,
			PowerUsage:  stat.PowerUsage,
			FanSpeed:    stat.FanSpeed,
		})
	}
	return systemGPUs
}

// Stat 获取系统统计信息
func (s *Service) Stat(ctx context.Context) *SystemStatResponse {
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

	return &SystemStatResponse{
		System:       detail,
		DiskIO:       diskIO,
		GPU:          buildSystemGPUs(ctx),
		Go:           goStat,
		Version:      config.Version,
		VersionCheck: s.CheckVersion(ctx),
	}
}

// Probe 探活
func (s *Service) Probe(ctx context.Context) *ProbeResponse {
	return &ProbeResponse{
		Agent:   map[string]bool{"available": config.Agent.BaseURL != "" && config.Agent.APIKey != ""},
		Apisix:  map[string]bool{"available": registry.IsApisixAvailable()},
		Docker:  map[string]bool{"available": registry.IsDockerAvailable(ctx)},
		Swarm:   map[string]bool{"available": registry.IsSwarmAvailable(ctx)},
		Compose: map[string]bool{"available": registry.IsComposeAvailable(ctx)},
	}
}

// Uptime 获取服务启动时间
func (s *Service) Uptime() *UptimeResponse {
	return &UptimeResponse{
		StartTime: startTime.Unix(),
		Uptime:    int64(time.Since(startTime).Seconds()),
	}
}
