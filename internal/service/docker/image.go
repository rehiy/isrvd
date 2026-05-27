package docker

import (
	"context"
	"fmt"

	pkgdocker "isrvd/pkgs/docker"
)

// ImageList 列出镜像
func (s *Service) ImageList(ctx context.Context, all bool) ([]*pkgdocker.ImageInfo, error) {
	list, err := s.docker.ImageList(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}
	return list, nil
}

// ImageAction 镜像操作
func (s *Service) ImageAction(ctx context.Context, req pkgdocker.ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("镜像ID不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.ImageAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("镜像操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// ImageTag 镜像打标签
func (s *Service) ImageTag(ctx context.Context, req pkgdocker.ImageTagRequest) error {
	if req.ID == "" {
		return fmt.Errorf("镜像ID不能为空")
	}
	if req.RepoTag == "" {
		return fmt.Errorf("目标标签不能为空")
	}
	if err := s.docker.ImageTag(ctx, req.ID, req.RepoTag); err != nil {
		return fmt.Errorf("镜像打标签失败: %w", err)
	}
	return nil
}

// ImageSearch 搜索镜像
func (s *Service) ImageSearch(ctx context.Context, term string) ([]*pkgdocker.ImageSearchResult, error) {
	if term == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	result, err := s.docker.ImageSearch(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("搜索镜像失败: %w", err)
	}
	return result, nil
}

// ImageBuild 构建镜像
func (s *Service) ImageBuild(ctx context.Context, req pkgdocker.ImageBuildRequest) (map[string]string, error) {
	if req.Tag == "" {
		return nil, fmt.Errorf("镜像标签不能为空")
	}
	if req.Dockerfile == "" {
		return nil, fmt.Errorf("Dockerfile 内容不能为空")
	}
	msg, err := s.docker.ImageBuild(ctx, req.Dockerfile, req.Tag)
	if err != nil {
		return nil, fmt.Errorf("构建镜像失败: %w", err)
	}
	return map[string]string{"tag": req.Tag, "message": msg}, nil
}

// ImagePrune 清理未使用的镜像
func (s *Service) ImagePrune(ctx context.Context, req pkgdocker.ImagePruneRequest) (*pkgdocker.ImagePruneReport, error) {
	report, err := s.docker.ImagePrune(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("清理镜像失败: %w", err)
	}
	return report, nil
}

// ImageInspect 获取镜像详情
func (s *Service) ImageInspect(ctx context.Context, id string) (*pkgdocker.ImageDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("镜像ID不能为空")
	}
	detail, err := s.docker.ImageInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取镜像详情失败: %w", err)
	}
	return detail, nil
}
