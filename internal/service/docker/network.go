package docker

import (
	"context"
	"fmt"

	pkgdocker "isrvd/pkgs/docker"
)

// ListNetworks 列出网络
func (s *Service) ListNetworks(ctx context.Context) (any, error) {
	return s.docker.ListNetworks(ctx)
}

// NetworkAction 网络操作
func (s *Service) NetworkAction(ctx context.Context, req pkgdocker.NetworkActionRequest) error {
	return s.docker.NetworkAction(ctx, req.ID, req.Action)
}

// CreateNetwork 创建网络
func (s *Service) CreateNetwork(ctx context.Context, req pkgdocker.NetworkCreateRequest) (map[string]string, error) {
	id, err := s.docker.CreateNetwork(ctx, req.Name, req.Driver)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "name": req.Name}, nil
}

// NetworkInspect 获取网络详情
func (s *Service) NetworkInspect(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("网络ID不能为空")
	}
	return s.docker.InspectNetwork(ctx, id)
}

// ListVolumes 列出卷
func (s *Service) ListVolumes(ctx context.Context) (any, error) {
	return s.docker.ListVolumes(ctx)
}

// VolumeAction 卷操作
func (s *Service) VolumeAction(ctx context.Context, req pkgdocker.VolumeActionRequest) error {
	return s.docker.VolumeAction(ctx, req.Name, req.Action)
}

// CreateVolume 创建卷
func (s *Service) CreateVolume(ctx context.Context, req pkgdocker.VolumeCreateRequest) (map[string]string, error) {
	name, mountpoint, err := s.docker.CreateVolume(ctx, req.Name, req.Driver)
	if err != nil {
		return nil, err
	}
	return map[string]string{"name": name, "mountpoint": mountpoint}, nil
}

// VolumeInspect 获取卷详情
func (s *Service) VolumeInspect(ctx context.Context, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("卷名称不能为空")
	}
	return s.docker.InspectVolume(ctx, name)
}
