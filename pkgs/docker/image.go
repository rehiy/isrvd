package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/rehiy/libgo/logman"
)

// ImageInfo Docker 镜像信息
type ImageInfo struct {
	ID          string   `json:"id"`
	ShortID     string   `json:"shortId"`
	RepoTags    []string `json:"repoTags"`
	RepoDigests []string `json:"repoDigests"`
	Size        int64    `json:"size"`
	Created     int64    `json:"created"`
}

// ImageList 列出镜像
func (s *DockerService) ImageList(ctx context.Context, all bool) ([]*ImageInfo, error) {
	images, err := s.client.ImageList(ctx, image.ListOptions{All: all})
	if err != nil {
		logman.Error("List images failed", "error", err)
		return nil, err
	}

	var result []*ImageInfo
	for _, img := range images {
		if !all && len(img.RepoTags) == 0 {
			continue
		}

		result = append(result, &ImageInfo{
			ID:          img.ID,
			ShortID:     ShortID(img.ID),
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Size:        img.Size,
			Created:     img.Created,
		})
	}

	return result, nil
}

// ImagePruneRequest 镜像清理请求
type ImagePruneRequest struct {
	// All 为 true 时清理所有未被任何容器使用的镜像（包括有标签的）；
	// 为 false 时仅清理悬空（未打标签）的镜像层，即 Docker 默认行为。
	All bool `json:"all"`
	// Until 仅清理在该时间之前创建的镜像，格式参考 Docker filters（如 "24h"、"2024-01-01T00:00:00"）。
	Until string `json:"until,omitempty"`
}

// ImagePruneReport 镜像清理结果
type ImagePruneReport struct {
	ImagesDeleted  []ImagePruneDeleted `json:"imagesDeleted"`
	SpaceReclaimed uint64              `json:"spaceReclaimed"`
}

// ImagePruneDeleted 单次删除条目（解除标签 / 删除层）
type ImagePruneDeleted struct {
	Untagged string `json:"untagged,omitempty"`
	Deleted  string `json:"deleted,omitempty"`
}

// ImagePrune 清理未使用的镜像。
// dangling=true 仅清理悬空层；dangling=false 等价于 `docker image prune -a`，
// 会回收所有未被容器引用的镜像（包括有标签但闲置的）。
func (s *DockerService) ImagePrune(ctx context.Context, req ImagePruneRequest) (*ImagePruneReport, error) {
	args := filters.NewArgs()
	// dangling=true 仅清理悬空层；dangling=false 清理所有未被容器引用的镜像
	args.Add("dangling", strconv.FormatBool(!req.All))
	if req.Until != "" {
		args.Add("until", req.Until)
	}

	report, err := s.client.ImagesPrune(ctx, args)
	if err != nil {
		logman.Error("Prune images failed", "all", req.All, "error", err)
		return nil, err
	}

	deleted := make([]ImagePruneDeleted, 0, len(report.ImagesDeleted))
	for _, d := range report.ImagesDeleted {
		deleted = append(deleted, ImagePruneDeleted{Untagged: d.Untagged, Deleted: d.Deleted})
	}

	logman.Info("Images pruned", "all", req.All, "deletedCount", len(deleted), "spaceReclaimed", report.SpaceReclaimed)
	return &ImagePruneReport{ImagesDeleted: deleted, SpaceReclaimed: report.SpaceReclaimed}, nil
}

// ImageAction 镜像操作
func (s *DockerService) ImageAction(ctx context.Context, id, action string) error {
	switch action {
	case "remove":
		deleted, err := s.client.ImageRemove(ctx, id, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			logman.Error("Remove image failed", "id", id, "error", err)
			return err
		}
		logman.Info("Image action performed", "action", action, "id", id, "deleted", len(deleted))
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	return nil
}

// ImageTagRequest 镜像标签请求
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`
	RepoTag string `json:"repoTag" binding:"required"`
}

// ImageTag 镜像打标签
func (s *DockerService) ImageTag(ctx context.Context, id, repoTag string) error {
	if err := s.client.ImageTag(ctx, id, repoTag); err != nil {
		logman.Error("Tag image failed", "id", id, "tag", repoTag, "error", err)
		return err
	}

	logman.Info("Image tagged", "id", id, "tag", repoTag)
	return nil
}

// ImageSearchResult 镜像搜索结果
type ImageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsOfficial  bool   `json:"isOfficial"`
	StarCount   int    `json:"starCount"`
}

// ImageSearch 搜索镜像
func (s *DockerService) ImageSearch(ctx context.Context, term string) ([]*ImageSearchResult, error) {
	results, err := s.client.ImageSearch(ctx, term, registry.SearchOptions{Limit: 25})
	if err != nil {
		logman.Error("Search image failed", "term", term, "error", err)
		return nil, err
	}

	var searchResults []*ImageSearchResult
	for _, r := range results {
		searchResults = append(searchResults, &ImageSearchResult{
			Name:        r.Name,
			Description: r.Description,
			IsOfficial:  r.IsOfficial,
			StarCount:   r.StarCount,
		})
	}

	return searchResults, nil
}

// ImageBuildRequest 镜像构建请求
type ImageBuildRequest struct {
	Dockerfile string `json:"dockerfile" binding:"required"`
	Tag        string `json:"tag"`
}

// ImageBuild 构建镜像
func (s *DockerService) ImageBuild(ctx context.Context, dockerfile, tag string) (string, error) {
	tarBuf, err := buildDockerfileTar(dockerfile)
	if err != nil {
		logman.Error("Build dockerfile tar failed", "error", err)
		return "", err
	}

	if tag == "" {
		tag = "custom:latest"
	}

	resp, err := s.client.ImageBuild(ctx, tarBuf, build.ImageBuildOptions{
		Tags: []string{tag},
	})
	if err != nil {
		logman.Error("Build image failed", "tag", tag, "error", err)
		return "", err
	}
	defer resp.Body.Close()

	var lastMessage string
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			logman.Error("Build image stream error", "tag", tag, "error", msg.Error)
			return "", errors.New(msg.Error)
		}
		if msg.Stream != "" {
			lastMessage = strings.TrimSpace(msg.Stream)
		}
	}

	logman.Info("Image built", "tag", tag)
	return lastMessage, nil
}

// ImageLayerInfo 镜像层信息
type ImageLayerInfo struct {
	Digest    string `json:"digest"`    // 层 digest（sha256:...）
	CreatedBy string `json:"createdBy"` // 构建命令
	Created   string `json:"created"`   // 创建时间
	Size      int64  `json:"size"`      // 层大小（字节），-1 表示空层
	Empty     bool   `json:"empty"`     // 是否为空层（无文件变更）
}

// ImageDetail 镜像详情响应
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

// ImageEnsure 确保镜像存在；若本地不存在则自动拉取。
// forcePull 为 true 时，无论本地是否存在都会重新拉取。
// 认证信息从 imageRef 的 host 自动匹配已配置的 registry。
func (s *DockerService) ImageEnsure(ctx context.Context, ref string, forcePull bool) error {
	if ref == "" {
		return nil
	}
	// 补全 tag
	imageRef := ref
	if !strings.Contains(imageRef, ":") && !strings.Contains(imageRef, "@") {
		imageRef += ":latest"
	}
	// forcePull=false 时，本地已存在则跳过
	if !forcePull {
		if _, err := s.client.ImageInspect(ctx, imageRef); err == nil {
			return nil
		}
	}
	logman.Info("Pulling image", "image", imageRef, "force", forcePull)
	if _, err := s.imagePull(ctx, imageRef); err != nil {
		return err
	}
	logman.Info("Image pulled successfully", "image", imageRef)
	return nil
}

// ImageConfig 获取镜像的原始运行配置（来自 Dockerfile 的默认值）
// 用于在从运行容器反推 compose 时过滤掉镜像内置的默认值
func (s *DockerService) ImageConfig(ctx context.Context, imageRef string) (*v1.DockerOCIImageConfig, error) {
	img, err := s.client.ImageInspect(ctx, imageRef)
	if err != nil {
		logman.Error("Get image config failed", "image", imageRef, "error", err)
		return nil, err
	}
	return img.Config, nil
}

// ImageInspect 获取镜像详情
func (s *DockerService) ImageInspect(ctx context.Context, id string) (*ImageDetail, error) {
	img, err := s.client.ImageInspect(ctx, id)
	if err != nil {
		logman.Error("Inspect image failed", "id", id, "error", err)
		return nil, err
	}

	// 提取暴露端口列表
	var exposedPorts []string
	if img.Config != nil {
		for port := range img.Config.ExposedPorts {
			exposedPorts = append(exposedPorts, port)
		}
	}

	// 统计层数
	layers := 0
	if img.RootFS.Type == "layers" {
		layers = len(img.RootFS.Layers)
	}

	// 获取层历史
	history, err := s.client.ImageHistory(ctx, id)
	if err != nil {
		logman.Warn("Get image history failed", "id", id, "error", err)
	}

	// 构建层详情：history 顺序为最新层在前，倒序后从底层开始编号
	var layerDetails []*ImageLayerInfo
	digestIdx := 0
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		cmd := strings.TrimPrefix(h.CreatedBy, "/bin/sh -c #(nop) ")
		isEmpty := h.Size == 0 && strings.Contains(h.CreatedBy, "#(nop)")
		info := &ImageLayerInfo{
			CreatedBy: cmd,
			Created:   time.Unix(h.Created, 0).UTC().Format(time.RFC3339),
			Empty:     isEmpty,
			Size:      h.Size,
		}
		if !isEmpty && digestIdx < len(img.RootFS.Layers) {
			info.Digest = img.RootFS.Layers[digestIdx]
			digestIdx++
		}
		layerDetails = append(layerDetails, info)
	}

	var cmd []string
	var entrypoint []string
	var env []string
	var workingDir string
	var labels map[string]string
	if img.Config != nil {
		cmd = img.Config.Cmd
		entrypoint = img.Config.Entrypoint
		env = img.Config.Env
		workingDir = img.Config.WorkingDir
		labels = img.Config.Labels
	}

	result := &ImageDetail{
		ID:           img.ID,
		ShortID:      ShortID(img.ID),
		RepoTags:     img.RepoTags,
		RepoDigests:  img.RepoDigests,
		Size:         img.Size,
		Created:      img.Created,
		Author:       img.Author,
		Architecture: img.Architecture,
		OS:           img.Os,
		Cmd:          cmd,
		Entrypoint:   entrypoint,
		Env:          env,
		WorkingDir:   workingDir,
		ExposedPorts: exposedPorts,
		Labels:       labels,
		Layers:       layers,
		LayerDetails: layerDetails,
	}

	return result, nil
}
