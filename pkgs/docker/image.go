package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/rehiy/libgo/logman"
)

// ImageList 列出镜像，直接返回 Docker SDK 原始列表项。
func (s *DockerService) ImageList(ctx context.Context, all bool) ([]image.Summary, error) {
	images, err := s.client.ImageList(ctx, image.ListOptions{All: all})
	if err != nil {
		logman.Error("List images failed", "error", err)
		return nil, err
	}
	return images, nil
}

// ImagePrune 清理未使用的镜像。
// dangling=true 仅清理悬空层；dangling=false 等价于 `docker image prune -a`，
// 会回收所有未被容器引用的镜像（包括有标签但闲置的）。
func (s *DockerService) ImagePrune(ctx context.Context, all bool, until string) (image.PruneReport, error) {
	args := filters.NewArgs()
	// dangling=true 仅清理悬空层；dangling=false 清理所有未被容器引用的镜像
	args.Add("dangling", strconv.FormatBool(!all))
	if until != "" {
		args.Add("until", until)
	}

	report, err := s.client.ImagesPrune(ctx, args)
	if err != nil {
		logman.Error("Prune images failed", "all", all, "error", err)
		return image.PruneReport{}, err
	}
	logman.Info("Images pruned", "all", all, "deletedCount", len(report.ImagesDeleted), "spaceReclaimed", report.SpaceReclaimed)
	return report, nil
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

// ImageTag 镜像打标签
func (s *DockerService) ImageTag(ctx context.Context, id, repoTag string) error {
	if err := s.client.ImageTag(ctx, id, repoTag); err != nil {
		logman.Error("Tag image failed", "id", id, "tag", repoTag, "error", err)
		return err
	}

	logman.Info("Image tagged", "id", id, "tag", repoTag)
	return nil
}

// ImageSearch 搜索镜像，直接返回 Docker SDK 原始搜索结果。
func (s *DockerService) ImageSearch(ctx context.Context, term string) ([]registry.SearchResult, error) {
	results, err := s.client.ImageSearch(ctx, term, registry.SearchOptions{Limit: 25})
	if err != nil {
		logman.Error("Search image failed", "term", term, "error", err)
		return nil, err
	}
	return results, nil
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

// ImageInspect 获取镜像原始详情及历史层信息。
func (s *DockerService) ImageInspect(ctx context.Context, id string) (*image.InspectResponse, []image.HistoryResponseItem, error) {
	img, err := s.client.ImageInspect(ctx, id)
	if err != nil {
		logman.Error("Inspect image failed", "id", id, "error", err)
		return nil, nil, err
	}

	history, err := s.client.ImageHistory(ctx, id)
	if err != nil {
		logman.Warn("Get image history failed", "id", id, "error", err)
	}
	return &img, history, nil
}
