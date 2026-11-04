package docker

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/shirou/gopsutil/v3/cpu"
)

// ContainerStatsResponse 容器统计信息响应，保持前端稳定响应结构。
type ContainerStatsResponse struct {
	Ts            int64                      `json:"ts"`            // 服务端采样时间（Unix 秒）
	ID            string                     `json:"id"`            // 容器 ID
	Name          string                     `json:"name"`          // 容器名称
	CPUPercent    float64                    `json:"cpuPercent"`    // CPU 使用率（百分比）
	CPUCores      int                        `json:"cpuCores"`      // 可用 CPU 核数
	CPUFreq       float64                    `json:"cpuFreq"`       // CPU 主频（MHz）
	MemoryUsage   int64                      `json:"memoryUsage"`   // 内存使用量（字节）
	MemoryLimit   int64                      `json:"memoryLimit"`   // 内存上限（字节）
	MemoryPercent float64                    `json:"memoryPercent"` // 内存使用率（百分比）
	NetworkRx     int64                      `json:"networkRx"`     // 网络接收总量（字节）
	NetworkTx     int64                      `json:"networkTx"`     // 网络发送总量（字节）
	BlockRead     int64                      `json:"blockRead"`     // 磁盘读取总量（字节）
	BlockWrite    int64                      `json:"blockWrite"`    // 磁盘写入总量（字节）
	Pids          int64                      `json:"pids"`          // 当前进程数
	PidsLimit     int64                      `json:"pidsLimit"`     // 进程数上限
	CPUThrottled  *CPUThrottledData          `json:"cpuThrottled"`  // CPU 节流明细
	NetworkDetail map[string]*StatsNetDetail `json:"networkDetail"` // 各网卡明细（按网卡名）
	BlockDetail   []*StatsBlockDetail        `json:"blockDetail"`   // 各块设备明细
	ProcessList   *ContainerProcessList      `json:"processList"`   // 容器内进程列表
}

// CPUThrottledData CPU 节流数据，保持前端稳定响应结构。
type CPUThrottledData struct {
	Periods          uint64 `json:"periods"`          // 调度周期总数
	ThrottledPeriods uint64 `json:"throttledPeriods"` // 被节流的周期数
	ThrottledTime    uint64 `json:"throttledTime"`    // 被节流的累计时间（纳秒）
}

// StatsNetDetail 网卡详细统计，保持前端稳定响应结构。
type StatsNetDetail struct {
	RxBytes   uint64 `json:"rxBytes"`   // 接收字节数
	RxPackets uint64 `json:"rxPackets"` // 接收包数
	RxErrors  uint64 `json:"rxErrors"`  // 接收错误数
	RxDropped uint64 `json:"rxDropped"` // 接收丢包数
	TxBytes   uint64 `json:"txBytes"`   // 发送字节数
	TxPackets uint64 `json:"txPackets"` // 发送包数
	TxErrors  uint64 `json:"txErrors"`  // 发送错误数
	TxDropped uint64 `json:"txDropped"` // 发送丢包数
}

// StatsBlockDetail 硬盘设备详细统计，保持前端稳定响应结构。
type StatsBlockDetail struct {
	Major uint64 `json:"major"` // 设备主号
	Minor uint64 `json:"minor"` // 设备次号
	Read  uint64 `json:"read"`  // 读取字节数
	Write uint64 `json:"write"` // 写入字节数
}

// ContainerProcessList 容器进程列表，保持前端稳定响应结构。
type ContainerProcessList struct {
	Titles    []string   `json:"titles"`    // 列标题（如 PID/USER/CMD）
	Processes [][]string `json:"processes"` // 进程行数据，每行与 titles 对应
}

// ContainerStats 获取容器统计信息
func (s *Service) ContainerStats(ctx context.Context, id string) (*ContainerStatsResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("容器ID不能为空")
	}
	stats, top, err := s.docker.ContainerStats(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取容器统计信息失败: %w", err)
	}
	return containerStatsResponse(id, stats, top), nil
}

// ─── 辅助函数 ───

func containerStatsResponse(id string, v container.StatsResponse, top *container.TopResponse) *ContainerStatsResponse {
	cpuCores := int(v.CPUStats.OnlineCPUs)
	if cpuCores == 0 {
		cpuCores = len(v.CPUStats.CPUUsage.PercpuUsage)
	}
	if cpuCores == 0 {
		cpuCores = 1
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(cpuCores) * 100.0
	}

	memCache, cgroupV1 := v.MemoryStats.Stats["total_inactive_file"]
	if !cgroupV1 {
		memCache = v.MemoryStats.Stats["inactive_file"]
	}
	memoryUsage := v.MemoryStats.Usage - min(memCache, v.MemoryStats.Usage)
	var memoryPercent float64
	if v.MemoryStats.Limit > 0 {
		memoryPercent = float64(memoryUsage) / float64(v.MemoryStats.Limit) * 100.0
	}

	var networkRx, networkTx int64
	networkDetail := make(map[string]*StatsNetDetail)
	for name, netStats := range v.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
		networkDetail[name] = &StatsNetDetail{
			RxBytes: netStats.RxBytes, RxPackets: netStats.RxPackets, RxErrors: netStats.RxErrors, RxDropped: netStats.RxDropped,
			TxBytes: netStats.TxBytes, TxPackets: netStats.TxPackets, TxErrors: netStats.TxErrors, TxDropped: netStats.TxDropped,
		}
	}

	var blockRead, blockWrite int64
	blockDetailMap := make(map[string]*StatsBlockDetail)
	for _, blkStats := range v.BlkioStats.IoServiceBytesRecursive {
		blkStats.Op = strings.ToLower(blkStats.Op)
		switch blkStats.Op {
		case "read":
			blockRead += int64(blkStats.Value)
		case "write":
			blockWrite += int64(blkStats.Value)
		}
		if blkStats.Op == "read" || blkStats.Op == "write" {
			key := fmt.Sprintf("%d:%d", blkStats.Major, blkStats.Minor)
			if _, ok := blockDetailMap[key]; !ok {
				blockDetailMap[key] = &StatsBlockDetail{Major: blkStats.Major, Minor: blkStats.Minor}
			}
			if blkStats.Op == "read" {
				blockDetailMap[key].Read += blkStats.Value
			} else {
				blockDetailMap[key].Write += blkStats.Value
			}
		}
	}
	blockDetail := slices.AppendSeq(make([]*StatsBlockDetail, 0, len(blockDetailMap)), maps.Values(blockDetailMap))
	slices.SortFunc(blockDetail, func(a, b *StatsBlockDetail) int {
		return cmp.Or(cmp.Compare(a.Major, b.Major), cmp.Compare(a.Minor, b.Minor))
	})

	name := strings.TrimPrefix(v.Name, "/")
	var processList *ContainerProcessList
	if top != nil {
		processList = &ContainerProcessList{Titles: top.Titles, Processes: top.Processes}
	}

	return &ContainerStatsResponse{
		Ts: time.Now().Unix(),
		ID: id, Name: name, CPUPercent: math.Round(cpuPercent*100) / 100, CPUCores: cpuCores, CPUFreq: math.Round(getCPUFreq()*100) / 100,
		MemoryUsage: int64(memoryUsage), MemoryLimit: int64(v.MemoryStats.Limit), MemoryPercent: math.Round(memoryPercent*100) / 100,
		NetworkRx: networkRx, NetworkTx: networkTx, BlockRead: blockRead, BlockWrite: blockWrite,
		Pids: int64(v.PidsStats.Current), PidsLimit: int64(v.PidsStats.Limit),
		CPUThrottled:  &CPUThrottledData{Periods: v.CPUStats.ThrottlingData.Periods, ThrottledPeriods: v.CPUStats.ThrottlingData.ThrottledPeriods, ThrottledTime: v.CPUStats.ThrottlingData.ThrottledTime},
		NetworkDetail: networkDetail, BlockDetail: blockDetail, ProcessList: processList,
	}
}

var (
	cpuFreqCache      float64
	cpuFreqMu         sync.Mutex
	cpuFreqLastUpdate time.Time
)

func getCPUFreq() float64 {
	cpuFreqMu.Lock()
	defer cpuFreqMu.Unlock()
	if time.Since(cpuFreqLastUpdate) < 5*time.Minute && cpuFreqCache > 0 {
		return cpuFreqCache
	}
	cpuFreqCache = 0
	if cpuInfos, err := cpu.Info(); err == nil && len(cpuInfos) > 0 {
		cpuFreqCache = cpuInfos[0].Mhz
	}
	cpuFreqLastUpdate = time.Now()
	return cpuFreqCache
}
