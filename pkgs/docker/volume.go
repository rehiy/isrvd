package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/rehiy/pango/logman"
)

// ListVolumes 列出卷
func (s *DockerService) ListVolumes(ctx context.Context) ([]*VolumeInfo, error) {
	volumes, err := s.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		logman.Error("List volumes failed", "error", err)
		return nil, err
	}

	var result []*VolumeInfo
	for _, vol := range volumes.Volumes {
		result = append(result, &VolumeInfo{
			Name: vol.Name, Driver: vol.Driver,
			Mountpoint: vol.Mountpoint, CreatedAt: vol.CreatedAt,
		})
	}

	return result, nil
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

// CreateVolume 创建卷
func (s *DockerService) CreateVolume(ctx context.Context, name, driver string) (string, string, error) {
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

// InspectVolume 获取卷详情
func (s *DockerService) InspectVolume(ctx context.Context, name string) (*VolumeInspectResponse, error) {
	volInfo, err := s.client.VolumeInspect(ctx, name)
	if err != nil {
		logman.Error("Volume inspect failed", "name", name, "error", err)
		return nil, err
	}

	// 查找使用此卷的容器
	var usedBy []*VolumeUsedByContainer
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		for _, ct := range containers {
			for _, mount := range ct.Mounts {
				if mount.Type == "volume" && mount.Name == name {
					ctName := ""
					if len(ct.Names) > 0 {
						ctName = strings.TrimPrefix(ct.Names[0], "/")
					}
					usedBy = append(usedBy, &VolumeUsedByContainer{
						ID:        ct.ID[:12],
						Name:      ctName,
						MountPath: mount.Destination,
						ReadOnly:  !mount.RW,
					})
				}
			}
		}
	}

	result := &VolumeInspectResponse{
		Name:       volInfo.Name,
		Driver:     volInfo.Driver,
		Mountpoint: volInfo.Mountpoint,
		CreatedAt:  volInfo.CreatedAt,
		Scope:      volInfo.Scope,
		UsedBy:     usedBy,
	}

	if volInfo.UsageData != nil {
		result.Size = volInfo.UsageData.Size
		result.RefCount = volInfo.UsageData.RefCount
	}

	return result, nil
}
