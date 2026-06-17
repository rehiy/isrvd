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
// ID 来自 URL path（:id），由 handler 在绑定前注入。
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`      // 源镜像 ID（来自 URL path）
	RepoTag string `json:"repoTag" binding:"required"` // 目标镜像标签（repo:tag）
}

// ImageBuildRequest 镜像构建请求。
type ImageBuildRequest struct {
	Dockerfile string `json:"dockerfile" binding:"required"` // Dockerfile 文本内容
	Tag        string `json:"tag"`                           // 构建产物的镜像标签
}

// ImagePruneRequest 镜像清理请求。
type ImagePruneRequest struct {
	All   bool   `json:"all"`             // true 清理所有未使用镜像，false 仅清理悬空镜像
	Until string `json:"until,omitempty"` // 仅清理早于该时间的镜像（如 "24h"）
}

// ImagePruneReport 镜像清理结果。
type ImagePruneReport struct {
	ImagesDeleted  []ImagePruneDeleted `json:"imagesDeleted"`  // 被删除的镜像条目
	SpaceReclaimed uint64              `json:"spaceReclaimed"` // 回收的磁盘空间（字节）
}

// ImagePruneDeleted 单次删除条目。
type ImagePruneDeleted struct {
	Untagged string `json:"untagged,omitempty"` // 被取消标签的镜像引用
	Deleted  string `json:"deleted,omitempty"`  // 被删除的镜像层 ID
}

// ImageInfo Docker 镜像信息，保持前端稳定响应结构。
type ImageInfo struct {
	ID          string   `json:"id"`          // 镜像完整 ID
	ShortID     string   `json:"shortId"`     // 镜像短 ID
	RepoTags    []string `json:"repoTags"`    // 仓库标签列表
	RepoDigests []string `json:"repoDigests"` // 仓库摘要列表
	Size        int64    `json:"size"`        // 镜像大小（字节）
	Created     int64    `json:"created"`     // 创建时间戳
}

// ImageSearchResult 镜像搜索结果，保持前端稳定响应结构。
type ImageSearchResult struct {
	Name        string `json:"name"`        // 镜像名称
	Description string `json:"description"` // 镜像描述
	IsOfficial  bool   `json:"isOfficial"`  // 是否官方镜像
	StarCount   int    `json:"starCount"`   // 星标数
}

// ImageLayerInfo 镜像层信息，保持前端稳定响应结构。
type ImageLayerInfo struct {
	Digest    string `json:"digest"`    // 层摘要
	CreatedBy string `json:"createdBy"` // 生成该层的构建指令
	Created   string `json:"created"`   // 层创建时间
	Size      int64  `json:"size"`      // 层大小（字节）
	Empty     bool   `json:"empty"`     // 是否为空层（无文件变更）
}

// ImageDetail 镜像详情响应，保持前端稳定响应结构。
type ImageDetail struct {
	ID           string            `json:"id"`           // 镜像完整 ID
	ShortID      string            `json:"shortId"`      // 镜像短 ID
	RepoTags     []string          `json:"repoTags"`     // 仓库标签列表
	RepoDigests  []string          `json:"repoDigests"`  // 仓库摘要列表
	Size         int64             `json:"size"`         // 镜像大小（字节）
	Created      string            `json:"created"`      // 创建时间
	Author       string            `json:"author"`       // 作者
	Architecture string            `json:"architecture"` // CPU 架构
	OS           string            `json:"os"`           // 操作系统
	Cmd          []string          `json:"cmd"`          // 默认启动命令
	Entrypoint   []string          `json:"entrypoint"`   // 入口点
	Env          []string          `json:"env"`          // 环境变量列表
	WorkingDir   string            `json:"workingDir"`   // 工作目录
	ExposedPorts []string          `json:"exposedPorts"` // 暴露的端口列表
	Labels       map[string]string `json:"labels"`       // 镜像标签
	Layers       int               `json:"layers"`       // 镜像层数
	LayerDetails []*ImageLayerInfo `json:"layerDetails"` // 各层详情
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
