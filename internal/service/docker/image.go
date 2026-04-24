package docker

import (
	"context"
	"fmt"

	pkgdocker "isrvd/pkgs/docker"
)

// ListImages 列出镜像
func (s *Service) ListImages(ctx context.Context, all bool) (any, error) {
	return s.docker.ListImages(ctx, all)
}

// ImageAction 镜像操作
func (s *Service) ImageAction(ctx context.Context, req pkgdocker.ImageActionRequest) error {
	return s.docker.ImageAction(ctx, req.ID, req.Action)
}

// TagImage 镜像打标签
func (s *Service) TagImage(ctx context.Context, req pkgdocker.ImageTagRequest) error {
	return s.docker.TagImage(ctx, req.ID, req.RepoTag)
}

// SearchImages 搜索镜像
func (s *Service) SearchImages(ctx context.Context, term string) (any, error) {
	if term == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	return s.docker.SearchImages(ctx, term)
}

// BuildImage 构建镜像
func (s *Service) BuildImage(ctx context.Context, req pkgdocker.ImageBuildRequest) (map[string]string, error) {
	msg, err := s.docker.BuildImage(ctx, req.Dockerfile, req.Tag)
	if err != nil {
		return nil, err
	}
	return map[string]string{"tag": req.Tag, "message": msg}, nil
}

// InspectImage 获取镜像详情
func (s *Service) InspectImage(ctx context.Context, id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("镜像ID不能为空")
	}
	return s.docker.InspectImage(ctx, id)
}
