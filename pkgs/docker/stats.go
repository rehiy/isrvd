package docker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/logman"
)

// ContainerStatsOnce 获取 Docker SDK 原始容器统计（单次采样，不含进程列表）。
func (s *DockerService) ContainerStatsOnce(ctx context.Context, id string) (container.StatsResponse, error) {
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

// ContainerStats 获取 Docker SDK 原始容器统计和进程列表。
func (s *DockerService) ContainerStats(ctx context.Context, id string) (container.StatsResponse, *container.TopResponse, error) {
	v, err := s.ContainerStatsOnce(ctx, id)
	if err != nil {
		return container.StatsResponse{}, nil, err
	}
	topResult, err := s.client.ContainerTop(ctx, id, nil)
	if err != nil {
		return v, nil, nil
	}
	return v, &topResult, nil
}
