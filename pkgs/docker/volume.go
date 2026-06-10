package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/rehiy/libgo/logman"
)

// VolumeList 列出卷，直接返回 Docker SDK 原始卷结构。
func (s *DockerService) VolumeList(ctx context.Context) ([]*volume.Volume, error) {
	volumes, err := s.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		logman.Error("List volumes failed", "error", err)
		return nil, err
	}
	return volumes.Volumes, nil
}

// VolumeAction 卷操作
func (s *DockerService) VolumeAction(ctx context.Context, name, action string) error {
	switch action {
	case "remove":
		if err := s.client.VolumeRemove(ctx, name, true); err != nil {
			logman.Error("Remove volume failed", "name", name, "error", err)
			return err
		}
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	logman.Info("Volume action performed", "action", action, "name", name)
	return nil
}

// VolumeCreate 创建卷
func (s *DockerService) VolumeCreate(ctx context.Context, name, driver string) (string, string, error) {
	if driver == "" {
		driver = "local"
	}

	resp, err := s.client.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: driver})
	if err != nil {
		logman.Error("Create volume failed", "error", err)
		return "", "", err
	}

	logman.Info("Volume created", "name", name)
	return resp.Name, resp.Mountpoint, nil
}

// VolumeInspect 获取卷原始详情。
func (s *DockerService) VolumeInspect(ctx context.Context, name string) (volume.Volume, error) {
	volInfo, err := s.client.VolumeInspect(ctx, name)
	if err != nil {
		logman.Error("Volume inspect failed", "name", name, "error", err)
		return volume.Volume{}, err
	}
	return volInfo, nil
}
