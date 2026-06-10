package docker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/logman"
)

// ContainerStats 获取 Docker SDK 原始容器统计和进程列表。
func (s *DockerService) ContainerStats(ctx context.Context, id string) (container.StatsResponse, *container.TopResponse, error) {
	stats, err := s.client.ContainerStats(ctx, id, false)
	if err != nil {
		logman.Error("Get container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, nil, err
	}
	defer stats.Body.Close()

	data, err := io.ReadAll(stats.Body)
	if err != nil {
		logman.Error("Read container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, nil, err
	}

	var v container.StatsResponse
	if err := json.Unmarshal(data, &v); err != nil {
		logman.Error("Parse container stats failed", "id", id, "error", err)
		return container.StatsResponse{}, nil, err
	}

	topResult, err := s.client.ContainerTop(ctx, id, nil)
	if err != nil {
		return v, nil, nil
	}
	return v, &topResult, nil
}
