package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/rehiy/pango/logman"
	"github.com/shirou/gopsutil/v3/cpu"
)

// CPU 主频缓存（不频繁变化，5分钟刷新一次）
var (
	cpuFreqCache      float64
	cpuFreqMu         sync.Mutex
	cpuFreqLastUpdate time.Time
)

// CPUThrottledData CPU 节流数据
type CPUThrottledData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttledPeriods"`
	ThrottledTime    uint64 `json:"throttledTime"`
}

// NetDetail 网卡详细统计
type NetDetail struct {
	RxBytes   uint64 `json:"rxBytes"`
	RxPackets uint64 `json:"rxPackets"`
	RxErrors  uint64 `json:"rxErrors"`
	RxDropped uint64 `json:"rxDropped"`
	TxBytes   uint64 `json:"txBytes"`
	TxPackets uint64 `json:"txPackets"`
	TxErrors  uint64 `json:"txErrors"`
	TxDropped uint64 `json:"txDropped"`
}

// BlockDetail 硬盘设备详细统计
type BlockDetail struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Read  uint64 `json:"read"`
	Write uint64 `json:"write"`
}

// ContainerProcessList 容器进程列表
type ContainerProcessList struct {
	Titles    []string   `json:"titles"`
	Processes [][]string `json:"processes"`
}

// ContainerStatsResponse 容器统计信息响应
type ContainerStatsResponse struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	CPUPercent    float64               `json:"cpuPercent"`
	CPUCores      int                   `json:"cpuCores"`
	CPUFreq       float64               `json:"cpuFreq"`
	MemoryUsage   int64                 `json:"memoryUsage"`
	MemoryLimit   int64                 `json:"memoryLimit"`
	MemoryPercent float64               `json:"memoryPercent"`
	NetworkRx     int64                 `json:"networkRx"`
	NetworkTx     int64                 `json:"networkTx"`
	BlockRead     int64                 `json:"blockRead"`
	BlockWrite    int64                 `json:"blockWrite"`
	Pids          int64                 `json:"pids"`
	PidsLimit     int64                 `json:"pidsLimit"`
	CPUThrottled  *CPUThrottledData     `json:"cpuThrottled"`
	NetworkDetail map[string]*NetDetail `json:"networkDetail"`
	BlockDetail   []*BlockDetail        `json:"blockDetail"`
	ProcessList   *ContainerProcessList `json:"processList"`
}

// GetContainerStats 获取容器统计信息
func (s *DockerService) GetContainerStats(ctx context.Context, id string) (*ContainerStatsResponse, error) {
	stats, err := s.client.ContainerStats(ctx, id, false)
	if err != nil {
		logman.Error("Get container stats failed", "id", id, "error", err)
		return nil, err
	}
	defer stats.Body.Close()

	data, err := io.ReadAll(stats.Body)
	if err != nil {
		logman.Error("Read container stats failed", "id", id, "error", err)
		return nil, err
	}

	var v types.StatsJSON
	if err := json.Unmarshal(data, &v); err != nil {
		logman.Error("Parse container stats failed", "id", id, "error", err)
		return nil, err
	}

	cpuCores := len(v.CPUStats.CPUUsage.PercpuUsage)

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(cpuCores) * 100.0
	}

	memoryUsage := v.MemoryStats.Usage - v.MemoryStats.Stats["cache"]
	var memoryPercent float64
	if v.MemoryStats.Limit > 0 {
		memoryPercent = float64(memoryUsage) / float64(v.MemoryStats.Limit) * 100.0
	}

	var networkRx, networkTx int64
	networkDetail := make(map[string]*NetDetail)
	for name, netStats := range v.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
		networkDetail[name] = &NetDetail{
			RxBytes:   netStats.RxBytes,
			RxPackets: netStats.RxPackets,
			RxErrors:  netStats.RxErrors,
			RxDropped: netStats.RxDropped,
			TxBytes:   netStats.TxBytes,
			TxPackets: netStats.TxPackets,
			TxErrors:  netStats.TxErrors,
			TxDropped: netStats.TxDropped,
		}
	}

	cpuThrottled := &CPUThrottledData{
		Periods:          v.CPUStats.ThrottlingData.Periods,
		ThrottledPeriods: v.CPUStats.ThrottlingData.ThrottledPeriods,
		ThrottledTime:    v.CPUStats.ThrottlingData.ThrottledTime,
	}

	var processList *ContainerProcessList
	topResult, err := s.client.ContainerTop(ctx, id, nil)
	if err == nil {
		processList = &ContainerProcessList{
			Titles:    topResult.Titles,
			Processes: topResult.Processes,
		}
	}

	var blockRead, blockWrite int64
	blockDetailMap := make(map[string]*BlockDetail)
	for _, blkStats := range v.BlkioStats.IoServiceBytesRecursive {
		switch blkStats.Op {
		case "read":
			blockRead += int64(blkStats.Value)
		case "write":
			blockWrite += int64(blkStats.Value)
		}
		if blkStats.Op == "read" || blkStats.Op == "write" {
			key := fmt.Sprintf("%d:%d", blkStats.Major, blkStats.Minor)
			if _, ok := blockDetailMap[key]; !ok {
				blockDetailMap[key] = &BlockDetail{
					Major: blkStats.Major,
					Minor: blkStats.Minor,
				}
			}
			if blkStats.Op == "read" {
				blockDetailMap[key].Read += blkStats.Value
			} else {
				blockDetailMap[key].Write += blkStats.Value
			}
		}
	}
	var blockDetail []*BlockDetail
	for _, detail := range blockDetailMap {
		blockDetail = append(blockDetail, detail)
	}
	sort.Slice(blockDetail, func(i, j int) bool {
		if blockDetail[i].Major != blockDetail[j].Major {
			return blockDetail[i].Major < blockDetail[j].Major
		}
		return blockDetail[i].Minor < blockDetail[j].Minor
	})

	name := ""
	if len(v.Name) > 0 {
		name = strings.TrimPrefix(v.Name, "/")
	}

	cpuFreq := getCpuFreq()

	result := &ContainerStatsResponse{
		ID:            id,
		Name:          name,
		CPUPercent:    math.Round(cpuPercent*100) / 100,
		CPUCores:      cpuCores,
		CPUFreq:       math.Round(cpuFreq*100) / 100,
		MemoryUsage:   int64(memoryUsage),
		MemoryLimit:   int64(v.MemoryStats.Limit),
		MemoryPercent: math.Round(memoryPercent*100) / 100,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		Pids:          int64(v.PidsStats.Current),
		PidsLimit:     int64(v.PidsStats.Limit),
		CPUThrottled:  cpuThrottled,
		NetworkDetail: networkDetail,
		BlockDetail:   blockDetail,
		ProcessList:   processList,
	}

	return result, nil
}

// getCpuFreq 获取 CPU 主频（使用 gopsutil，跨平台兼容）
func getCpuFreq() float64 {
	cpuFreqMu.Lock()
	defer cpuFreqMu.Unlock()

	if time.Since(cpuFreqLastUpdate) < 5*time.Minute && cpuFreqCache > 0 {
		return cpuFreqCache
	}

	cpuFreqCache = 0
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		cpuFreqCache = cpuInfos[0].Mhz
	}

	cpuFreqLastUpdate = time.Now()
	return cpuFreqCache
}
