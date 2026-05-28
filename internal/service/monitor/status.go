package monitor

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/rehiy/libgo/gpu"
	"github.com/rehiy/libgo/psutil"
	"github.com/shirou/gopsutil/v3/disk"

	"isrvd/config"
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

// HostStat 主机监控采集数据
type HostStat struct {
	System  *psutil.DetailStat `json:"system"`
	DiskIO  []*DiskIOStat      `json:"diskIO"`
	GPU     []*SystemGPU       `json:"gpu"`
	Go      *GoRuntimeStat     `json:"go"`
	Version string             `json:"version"`
}

// CollectHostStat 采集主机监控数据
func CollectHostStat(ctx context.Context) *HostStat {
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

	return &HostStat{
		System:  detail,
		DiskIO:  diskIO,
		GPU:     buildSystemGPUs(ctx),
		Go:      goStat,
		Version: config.Version,
	}
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
	deviceStats, err := gpu.GetGPUStats(ctx)
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
