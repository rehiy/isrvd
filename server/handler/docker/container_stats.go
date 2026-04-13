package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"
	"github.com/shirou/gopsutil/v3/cpu"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// CPU 主频缓存（不频繁变化，5分钟刷新一次）
var (
	cpuFreqCache      float64
	cpuFreqMu         sync.Mutex
	cpuFreqLastUpdate time.Time
)

// getCpuFreq 获取 CPU 主频（使用 gopsutil，跨平台兼容）
func getCpuFreq() float64 {
	cpuFreqMu.Lock()
	defer cpuFreqMu.Unlock()

	// 5分钟内使用缓存
	if time.Since(cpuFreqLastUpdate) < 5*time.Minute && cpuFreqCache > 0 {
		return cpuFreqCache
	}

	cpuFreqCache = 0

	// 使用 gopsutil 获取 CPU 主频
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		cpuFreqCache = cpuInfos[0].Mhz
	}

	cpuFreqLastUpdate = time.Now()
	return cpuFreqCache
}

// ContainerStats 获取容器统计信息
func (h *DockerHandler) ContainerStats(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		logman.Error("Container stats failed", "error", "container ID is empty")
		helper.RespondError(c, http.StatusBadRequest, "容器 ID 不能为空")
		return
	}

	ctx := c.Request.Context()

	stats, err := h.dockerClient.ContainerStats(ctx, id, false)
	if err != nil {
		logman.Error("Get container stats failed", "id", id, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取容器统计信息失败: "+err.Error())
		return
	}
	defer stats.Body.Close()

	data, err := io.ReadAll(stats.Body)
	if err != nil {
		logman.Error("Read container stats failed", "id", id, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "读取统计信息失败")
		return
	}

	var v types.StatsJSON
	if err := json.Unmarshal(data, &v); err != nil {
		logman.Error("Parse container stats failed", "id", id, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "解析统计信息失败")
		return
	}

	// 从 Docker Stats API 获取 CPU 核心数
	cpuCores := len(v.CPUStats.CPUUsage.PercpuUsage)

	// 计算 CPU 使用率
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(cpuCores) * 100.0
	}

	// 计算内存使用率
	memoryUsage := v.MemoryStats.Usage - v.MemoryStats.Stats["cache"]
	var memoryPercent float64
	if v.MemoryStats.Limit > 0 {
		memoryPercent = float64(memoryUsage) / float64(v.MemoryStats.Limit) * 100.0
	}

	// 计算网络 I/O
	var networkRx, networkTx int64
	networkDetail := make(map[string]*model.NetDetail)
	for name, netStats := range v.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
		networkDetail[name] = &model.NetDetail{
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

	// CPU 节流数据
	cpuThrottled := &model.CpuThrottledData{
		Periods:          v.CPUStats.ThrottlingData.Periods,
		ThrottledPeriods: v.CPUStats.ThrottlingData.ThrottledPeriods,
		ThrottledTime:    v.CPUStats.ThrottlingData.ThrottledTime,
	}

	// 获取进程列表
	var processList *model.ContainerProcessList
	topResult, err := h.dockerClient.ContainerTop(ctx, id, nil)
	if err == nil {
		processList = &model.ContainerProcessList{
			Titles:    topResult.Titles,
			Processes: topResult.Processes,
		}
	}

	// 计算磁盘 I/O
	var blockRead, blockWrite int64
	blockDetailMap := make(map[string]*model.BlockDetail)
	for _, blkStats := range v.BlkioStats.IoServiceBytesRecursive {
		switch blkStats.Op {
		case "read":
			blockRead += int64(blkStats.Value)
		case "write":
			blockWrite += int64(blkStats.Value)
		}
		// 按设备聚合详情
		if blkStats.Op == "read" || blkStats.Op == "write" {
			key := fmt.Sprintf("%d:%d", blkStats.Major, blkStats.Minor)
			if _, ok := blockDetailMap[key]; !ok {
				blockDetailMap[key] = &model.BlockDetail{
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
	// 转换为有序列表
	var blockDetail []*model.BlockDetail
	for _, detail := range blockDetailMap {
		blockDetail = append(blockDetail, detail)
	}
	// 按主设备号排序
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

	// 获取 CPU 主频（使用缓存）
	cpuFreq := getCpuFreq()

	result := model.ContainerStatsResponse{
		ID:            id,
		Name:          name,
		CPUPercent:    math.Round(cpuPercent*100) / 100,
		CpuCores:      cpuCores,
		CpuFreq:       math.Round(cpuFreq*100) / 100,
		MemoryUsage:   int64(memoryUsage),
		MemoryLimit:   int64(v.MemoryStats.Limit),
		MemoryPercent: math.Round(memoryPercent*100) / 100,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		Pids:          int64(v.PidsStats.Current),
		PidsLimit:     int64(v.PidsStats.Limit),
		CpuThrottled:  cpuThrottled,
		NetworkDetail: networkDetail,
		BlockDetail:   blockDetail,
		ProcessList:   processList,
	}

	helper.RespondSuccess(c, "Container stats retrieved successfully", result)
}
