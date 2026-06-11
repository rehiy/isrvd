package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"

	pkgDocker "isrvd/pkgs/docker"
)

// ImageTagRequest 镜像标签请求。
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`
	RepoTag string `json:"repoTag" binding:"required"`
}

// ImageBuildRequest 镜像构建请求。
type ImageBuildRequest struct {
	Dockerfile string `json:"dockerfile" binding:"required"`
	Tag        string `json:"tag"`
}

// ImagePruneRequest 镜像清理请求。
type ImagePruneRequest struct {
	All   bool   `json:"all"`
	Until string `json:"until,omitempty"`
}

// ImagePruneReport 镜像清理结果。
type ImagePruneReport struct {
	ImagesDeleted  []ImagePruneDeleted `json:"imagesDeleted"`
	SpaceReclaimed uint64              `json:"spaceReclaimed"`
}

// ImagePruneDeleted 单次删除条目。
type ImagePruneDeleted struct {
	Untagged string `json:"untagged,omitempty"`
	Deleted  string `json:"deleted,omitempty"`
}

// ImageInfo Docker 镜像信息，保持前端稳定响应结构。
type ImageInfo struct {
	ID          string   `json:"id"`
	ShortID     string   `json:"shortId"`
	RepoTags    []string `json:"repoTags"`
	RepoDigests []string `json:"repoDigests"`
	Size        int64    `json:"size"`
	Created     int64    `json:"created"`
}

// ImageSearchResult 镜像搜索结果，保持前端稳定响应结构。
type ImageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsOfficial  bool   `json:"isOfficial"`
	StarCount   int    `json:"starCount"`
}

// ImageLayerInfo 镜像层信息，保持前端稳定响应结构。
type ImageLayerInfo struct {
	Digest    string `json:"digest"`
	CreatedBy string `json:"createdBy"`
	Created   string `json:"created"`
	Size      int64  `json:"size"`
	Empty     bool   `json:"empty"`
}

// ImageDetail 镜像详情响应，保持前端稳定响应结构。
type ImageDetail struct {
	ID           string            `json:"id"`
	ShortID      string            `json:"shortId"`
	RepoTags     []string          `json:"repoTags"`
	RepoDigests  []string          `json:"repoDigests"`
	Size         int64             `json:"size"`
	Created      string            `json:"created"`
	Author       string            `json:"author"`
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	Cmd          []string          `json:"cmd"`
	Entrypoint   []string          `json:"entrypoint"`
	Env          []string          `json:"env"`
	WorkingDir   string            `json:"workingDir"`
	ExposedPorts []string          `json:"exposedPorts"`
	Labels       map[string]string `json:"labels"`
	Layers       int               `json:"layers"`
	LayerDetails []*ImageLayerInfo `json:"layerDetails"`
}

// ImageList 列出镜像
func (s *Service) ImageList(ctx context.Context, all bool) ([]*ImageInfo, error) {
	list, err := s.docker.ImageList(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}
	result := make([]*ImageInfo, 0, len(list))
	for _, img := range list {
		if !all && len(img.RepoTags) == 0 {
			continue
		}
		result = append(result, &ImageInfo{
			ID:          img.ID,
			ShortID:     pkgDocker.ShortID(img.ID),
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Size:        img.Size,
			Created:     img.Created,
		})
	}
	return result, nil
}

// ImageAction 镜像操作
func (s *Service) ImageAction(ctx context.Context, req ActionRequest) error {
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
func (s *Service) ImageTag(ctx context.Context, req ImageTagRequest) error {
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
func (s *Service) ImageSearch(ctx context.Context, term string) ([]*ImageSearchResult, error) {
	if term == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	results, err := s.docker.ImageSearch(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("搜索镜像失败: %w", err)
	}
	return imageSearchResults(results), nil
}

// ImageBuild 构建镜像
func (s *Service) ImageBuild(ctx context.Context, req ImageBuildRequest) (map[string]string, error) {
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
func (s *Service) ImagePrune(ctx context.Context, req ImagePruneRequest) (*ImagePruneReport, error) {
	report, err := s.docker.ImagePrune(ctx, req.All, req.Until)
	if err != nil {
		return nil, fmt.Errorf("清理镜像失败: %w", err)
	}
	deleted := make([]ImagePruneDeleted, 0, len(report.ImagesDeleted))
	for _, d := range report.ImagesDeleted {
		deleted = append(deleted, ImagePruneDeleted{Untagged: d.Untagged, Deleted: d.Deleted})
	}
	return &ImagePruneReport{ImagesDeleted: deleted, SpaceReclaimed: report.SpaceReclaimed}, nil
}

// ImageInspect 获取镜像详情
func (s *Service) ImageInspect(ctx context.Context, id string) (*ImageDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("镜像ID不能为空")
	}
	img, history, err := s.docker.ImageInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取镜像详情失败: %w", err)
	}
	return imageDetail(img, history), nil
}

func imageSearchResults(results []registry.SearchResult) []*ImageSearchResult {
	out := make([]*ImageSearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, &ImageSearchResult{Name: r.Name, Description: r.Description, IsOfficial: r.IsOfficial, StarCount: r.StarCount})
	}
	return out
}

func imageDetail(img *image.InspectResponse, history []image.HistoryResponseItem) *ImageDetail {
	if img == nil {
		return nil
	}
	var exposedPorts []string
	var cmd []string
	var entrypoint []string
	var env []string
	var workingDir string
	var labels map[string]string
	if img.Config != nil {
		for port := range img.Config.ExposedPorts {
			exposedPorts = append(exposedPorts, port)
		}
		cmd = img.Config.Cmd
		entrypoint = img.Config.Entrypoint
		env = img.Config.Env
		workingDir = img.Config.WorkingDir
		labels = img.Config.Labels
	}
	layers := 0
	if img.RootFS.Type == "layers" {
		layers = len(img.RootFS.Layers)
	}
	layerDetails := imageLayerDetails(img, history)
	return &ImageDetail{
		ID: img.ID, ShortID: pkgDocker.ShortID(img.ID), RepoTags: img.RepoTags, RepoDigests: img.RepoDigests,
		Size: img.Size, Created: img.Created, Author: img.Author, Architecture: img.Architecture, OS: img.Os,
		Cmd: cmd, Entrypoint: entrypoint, Env: env, WorkingDir: workingDir, ExposedPorts: exposedPorts, Labels: labels,
		Layers: layers, LayerDetails: layerDetails,
	}
}

func imageLayerDetails(img *image.InspectResponse, history []image.HistoryResponseItem) []*ImageLayerInfo {
	if img == nil {
		return nil
	}
	details := make([]*ImageLayerInfo, 0, len(history))
	digestIdx := 0
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		cmd := strings.TrimPrefix(h.CreatedBy, "/bin/sh -c #(nop) ")
		isEmpty := h.Size == 0 && strings.Contains(h.CreatedBy, "#(nop)")
		info := &ImageLayerInfo{CreatedBy: cmd, Created: time.Unix(h.Created, 0).UTC().Format(time.RFC3339), Empty: isEmpty, Size: h.Size}
		if !isEmpty && digestIdx < len(img.RootFS.Layers) {
			info.Digest = img.RootFS.Layers[digestIdx]
			digestIdx++
		}
		details = append(details, info)
	}
	return details
}
