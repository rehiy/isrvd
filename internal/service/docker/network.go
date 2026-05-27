package docker

import (
	"context"
	"fmt"

	pkgdocker "isrvd/pkgs/docker"
)

// NetworkList 列出网络
func (s *Service) NetworkList(ctx context.Context) ([]*pkgdocker.NetworkInfo, error) {
	list, err := s.docker.NetworkList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取网络列表失败: %w", err)
	}
	return list, nil
}

// NetworkAction 网络操作
func (s *Service) NetworkAction(ctx context.Context, req pkgdocker.ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("网络ID不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.NetworkAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("网络操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// NetworkCreate 创建网络
func (s *Service) NetworkCreate(ctx context.Context, req pkgdocker.NetworkSpec) (map[string]string, error) {
	id, err := s.docker.NetworkCreate(ctx, req.Name, req.Driver, req.Subnet)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "name": req.Name}, nil
}

// NetworkInspect 获取网络详情
func (s *Service) NetworkInspect(ctx context.Context, id string) (*pkgdocker.NetworkDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("网络ID不能为空")
	}
	return s.docker.NetworkInspect(ctx, id)
}

// VolumeList 列出卷
func (s *Service) VolumeList(ctx context.Context) ([]*pkgdocker.VolumeInfo, error) {
	list, err := s.docker.VolumeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取数据卷列表失败: %w", err)
	}
	return list, nil
}

// VolumeAction 卷操作（ID 字段即卷名）
func (s *Service) VolumeAction(ctx context.Context, req pkgdocker.ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("数据卷名称不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.VolumeAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("数据卷操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// VolumeCreate 创建卷
func (s *Service) VolumeCreate(ctx context.Context, req pkgdocker.VolumeSpec) (map[string]string, error) {
	name, mountpoint, err := s.docker.VolumeCreate(ctx, req.Name, req.Driver)
	if err != nil {
		return nil, err
	}
	return map[string]string{"name": name, "mountpoint": mountpoint}, nil
}

// VolumeInspect 获取卷详情
func (s *Service) VolumeInspect(ctx context.Context, name string) (*pkgdocker.VolumeDetail, error) {
	if name == "" {
		return nil, fmt.Errorf("卷名称不能为空")
	}
	return s.docker.VolumeInspect(ctx, name)
}
