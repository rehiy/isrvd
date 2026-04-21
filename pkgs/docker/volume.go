package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/rehiy/pango/logman"
)

// VolumeInfo Docker 卷信息
type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"createdAt"`
	Size       int64  `json:"size"`
}

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

// VolumeActionRequest 卷操作请求
type VolumeActionRequest struct {
	Name   string `json:"name" binding:"required"`
	Action string `json:"action" binding:"required"`
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

// VolumeCreateRequest 创建卷请求
type VolumeCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"`
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

// VolumeUsedByContainer 使用卷的容器信息
type VolumeUsedByContainer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

// VolumeInspectResponse 数据卷详情响应
type VolumeInspectResponse struct {
	Name       string                   `json:"name"`
	Driver     string                   `json:"driver"`
	Mountpoint string                   `json:"mountpoint"`
	CreatedAt  string                   `json:"createdAt"`
	Scope      string                   `json:"scope"`
	Size       int64                    `json:"size"`
	RefCount   int64                    `json:"refCount"`
	UsedBy     []*VolumeUsedByContainer `json:"usedBy"`
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
	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
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
