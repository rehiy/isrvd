package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/build"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/rehiy/pango/logman"
)

// ImageInfo Docker 镜像信息
type ImageInfo struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"shortId"`
	RepoTags []string `json:"repoTags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
}

// ListImages 列出镜像
func (s *DockerService) ListImages(ctx context.Context, all bool) ([]*ImageInfo, error) {
	images, err := s.client.ImageList(ctx, dockerimage.ListOptions{All: all})
	if err != nil {
		logman.Error("List images failed", "error", err)
		return nil, err
	}

	var result []*ImageInfo
	for _, img := range images {
		if !all && len(img.RepoTags) == 0 {
			continue
		}

		id := img.ID
		shortID := id
		if len(id) > 7 && strings.HasPrefix(id, "sha256:") {
			shortID = id[7:min(19, len(id))]
		} else if len(id) > 12 {
			shortID = id[:12]
		}
		result = append(result, &ImageInfo{
			ID: id, ShortID: shortID, RepoTags: img.RepoTags,
			Size: img.Size, Created: img.Created,
		})
	}

	return result, nil
}

// ImageActionRequest 镜像操作请求
type ImageActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// ImageAction 镜像操作
func (s *DockerService) ImageAction(ctx context.Context, id, action string) error {
	switch action {
	case "remove":
		_, err := s.client.ImageRemove(ctx, id, dockerimage.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			logman.Error("Remove image failed", "id", id, "error", err)
			return err
		}
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	logman.Info("Image action performed", "action", action, "id", id)
	return nil
}

// ImageTagRequest 镜像标签请求
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`
	RepoTag string `json:"repoTag" binding:"required"`
}

// TagImage 镜像打标签
func (s *DockerService) TagImage(ctx context.Context, id, repoTag string) error {
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
	IsAutomated bool   `json:"isAutomated"`
	StarCount   int    `json:"starCount"`
}

// SearchImages 搜索镜像
func (s *DockerService) SearchImages(ctx context.Context, term string) ([]*ImageSearchResult, error) {
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
			IsAutomated: r.IsAutomated,
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

// BuildImage 构建镜像
func (s *DockerService) BuildImage(ctx context.Context, dockerfile, tag string) (string, error) {
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

// ImageInspectResponse 镜像详情响应
type ImageInspectResponse struct {
	ID           string            `json:"id"`
	ShortID      string            `json:"shortId"`
	RepoTags     []string          `json:"repoTags"`
	RepoDigests  []string          `json:"repoDigests"`
	Size         int64             `json:"size"`
	VirtualSize  int64             `json:"virtualSize"`
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

// EnsureImage 确保镜像存在；若本地不存在则自动从 daemon 配置的 mirror 拉取。
// 拉取时不携带认证信息，依赖 daemon 的 mirror/proxy 配置。
func (s *DockerService) EnsureImage(ctx context.Context, ref string) error {
	if ref == "" {
		return nil
	}
	// 补全 tag
	imageRef := ref
	if !strings.Contains(imageRef, ":") && !strings.Contains(imageRef, "@") {
		imageRef += ":latest"
	}
	// 检查本地是否已存在
	_, _, err := s.client.ImageInspectWithRaw(ctx, imageRef)
	if err == nil {
		return nil // 已存在，无需拉取
	}
	logman.Info("Image not found locally, pulling", "image", imageRef)
	reader, err := s.client.ImagePull(ctx, imageRef, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("拉取镜像 %s 失败: %w", imageRef, err)
	}
	defer reader.Close()
	// 消费响应流，等待拉取完成
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			return fmt.Errorf("拉取镜像 %s 失败: %s", imageRef, msg.Error)
		}
	}
	logman.Info("Image pulled successfully", "image", imageRef)
	return nil
}

// InspectImage 获取镜像详情
func (s *DockerService) InspectImage(ctx context.Context, id string) (*ImageInspectResponse, error) {
	img, _, err := s.client.ImageInspectWithRaw(ctx, id)
	if err != nil {
		logman.Error("Inspect image failed", "id", id, "error", err)
		return nil, err
	}

	shortID := img.ID
	if strings.HasPrefix(shortID, "sha256:") && len(shortID) > 19 {
		shortID = shortID[7:19]
	}

	// 提取暴露端口列表
	var exposedPorts []string
	for port := range img.Config.ExposedPorts {
		exposedPorts = append(exposedPorts, string(port))
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
			Created:   formatUnixTime(h.Created),
			Empty:     isEmpty,
			Size:      h.Size,
		}
		if !isEmpty && digestIdx < len(img.RootFS.Layers) {
			info.Digest = img.RootFS.Layers[digestIdx]
			digestIdx++
		}
		layerDetails = append(layerDetails, info)
	}

	result := &ImageInspectResponse{
		ID:           img.ID,
		ShortID:      shortID,
		RepoTags:     img.RepoTags,
		RepoDigests:  img.RepoDigests,
		Size:         img.Size,
		VirtualSize:  img.VirtualSize,
		Created:      img.Created,
		Author:       img.Author,
		Architecture: img.Architecture,
		OS:           img.Os,
		Cmd:          img.Config.Cmd,
		Entrypoint:   img.Config.Entrypoint,
		Env:          img.Config.Env,
		WorkingDir:   img.Config.WorkingDir,
		ExposedPorts: exposedPorts,
		Labels:       img.Config.Labels,
		Layers:       layers,
		LayerDetails: layerDetails,
	}

	return result, nil
}
