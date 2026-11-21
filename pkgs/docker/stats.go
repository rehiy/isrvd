package docker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/logman"
)

// ContainerStats 获取 Docker SDK 原始容器统计，不查询进程列表。
func (s *DockerService) ContainerStats(ctx context.Context, id string) (container.StatsResponse, error) {
	stats, err := s.client.ContainerStats(ctx, id, false)
	if err != nil {
		logman.Error("Get container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, err
	}
	defer stats.Body.Close()

	data, err := io.ReadAll(stats.Body)
	if err != nil {
		logman.Error("Read container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, err
	}

	var v container.StatsResponse
	if err := json.Unmarshal(data, &v); err != nil {
		logman.Error("Parse container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, err
	}

	return v, nil
}

// ContainerProcessList 获取 Docker SDK 原始容器进程列表。
func (s *DockerService) ContainerProcessList(ctx context.Context, id string) (*container.TopResponse, error) {
	top, err := s.client.ContainerTop(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return &top, nil
}

// ContainerMemoryUsage 返回扣除非活跃文件缓存的内存用量，兼容 cgroup v1/v2。
func ContainerMemoryUsage(stats container.MemoryStats) uint64 {
	cache, cgroupV1 := stats.Stats["total_inactive_file"]
	if !cgroupV1 {
		cache = stats.Stats["inactive_file"]
	}
	return stats.Usage - min(cache, stats.Usage)
}
