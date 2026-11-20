package node

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// OverviewInfo 是中控全局概览的按需快照，节点故障不会影响其他节点的结果。
type OverviewInfo struct {
	CollectedAt int64               `json:"collectedAt"`
	Online      int                 `json:"online"`
	Nodes       []*OverviewNodeInfo `json:"nodes"`
}

// OverviewNodeInfo 仅包含概览所需的节点元数据，不暴露端口映射或认证信息。
type OverviewNodeInfo struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Hostname     string              `json:"hostname"`
	Status       string              `json:"status"`
	OS           string              `json:"os"`
	Arch         string              `json:"arch"`
	AgentVersion string              `json:"agentVersion,omitempty"`
	Online       bool                `json:"online"`
	Latency      int64               `json:"latency"`
	System       *OverviewSystemInfo `json:"system,omitempty"`
	Docker       *OverviewDockerInfo `json:"docker,omitempty"`
	Swarm        *OverviewSwarmInfo  `json:"swarm,omitempty"`
	MonitorError bool                `json:"monitorError,omitempty"`
}

type OverviewSystemInfo struct {
	CollectedAt int64    `json:"collectedAt"`
	CPUPercent  *float64 `json:"cpuPercent,omitempty"`
	MemoryUsed  uint64   `json:"memoryUsed"`
	MemoryTotal uint64   `json:"memoryTotal"`
	DiskUsed    uint64   `json:"diskUsed"`
	DiskTotal   uint64   `json:"diskTotal"`
}

type OverviewDockerInfo struct {
	ContainersRunning int `json:"containersRunning"`
	ContainersStopped int `json:"containersStopped"`
	ContainersPaused  int `json:"containersPaused"`
}

type OverviewSwarmInfo struct {
	Services int `json:"services"`
	Nodes    int `json:"nodes"`
}

// OverviewStats 并发获取在线节点的现有只读指标，最多同时访问 6 个受管节点。
// 不持久化快照，不等待离线节点；每个节点的采集总预算为 12 秒。
func (s *Service) OverviewStats(ctx context.Context) *OverviewInfo {
	nodes := s.NodeList()
	result := &OverviewInfo{Nodes: make([]*OverviewNodeInfo, len(nodes))}
	limit := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, n := range nodes {
		item := &OverviewNodeInfo{
			ID: n.ID, Name: n.Name, Hostname: n.Hostname, Status: n.Status,
			OS: n.OS, Arch: n.Arch, AgentVersion: n.AgentVersion,
			Online: n.Online, Latency: n.Latency,
		}
		result.Nodes[i] = item
		if !n.Online {
			continue
		}
		result.Online++
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				return
			}
			s.overviewNodeStats(ctx, item)
		}()
	}
	wg.Wait()
	result.CollectedAt = time.Now().Unix()
	return result
}

// overviewNodeStats 复用受管机已有业务接口，失败时仅该节点的数据为空。
func (s *Service) overviewNodeStats(ctx context.Context, item *OverviewNodeInfo) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	var record struct {
		Ts   int64 `json:"ts"`
		Data struct {
			System *struct {
				CPUPercent  []float64 `json:"cpuPercent"`
				MemoryUsed  uint64    `json:"memoryUsed"`
				MemoryTotal uint64    `json:"memoryTotal"`
				DiskUsed    uint64    `json:"diskUsed"`
				DiskTotal   uint64    `json:"diskTotal"`
			} `json:"system"`
		} `json:"data"`
	}
	if err := s.overviewFetch(ctx, item.ID, "/api/overview/monitor?since=0&type=host", &record); err == nil && record.Data.System != nil {
		metric := record.Data.System
		stat := &OverviewSystemInfo{
			CollectedAt: record.Ts, MemoryUsed: metric.MemoryUsed, MemoryTotal: metric.MemoryTotal,
			DiskUsed: metric.DiskUsed, DiskTotal: metric.DiskTotal,
		}
		if len(metric.CPUPercent) > 0 {
			var total float64
			for _, percent := range metric.CPUPercent {
				total += percent
			}
			average := total / float64(len(metric.CPUPercent))
			stat.CPUPercent = &average
		}
		item.System = stat
	} else {
		item.MonitorError = true
	}

	var docker OverviewDockerInfo
	if err := s.overviewFetch(ctx, item.ID, "/api/docker/info", &docker); err == nil {
		item.Docker = &docker
	}
	var swarm OverviewSwarmInfo
	if err := s.overviewFetch(ctx, item.ID, "/api/swarm/info", &swarm); err == nil {
		item.Swarm = &swarm
	}
}

// overviewFetch 通过节点隧道请求已有接口，不透传上游错误或未筛选的业务响应。
func (s *Service) overviewFetch(ctx context.Context, id, path string, out any) error {
	body, err := s.Fetch(ctx, id, path)
	if err != nil {
		return err
	}
	var envelope struct {
		Success bool            `json:"success"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if !envelope.Success || len(envelope.Payload) == 0 || string(envelope.Payload) == "null" {
		return fmt.Errorf("节点指标不可用")
	}
	return json.Unmarshal(envelope.Payload, out)
}
