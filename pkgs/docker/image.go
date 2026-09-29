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

// ImagePush 推送镜像到仓库
func (s *DockerService) ImagePush(ctx context.Context, imageRef, registryURL, namespace string) (string, string, error) {
	// 提取镜像的短名称
	imageName := imageRef
	if idx := strings.LastIndex(imageName, "/"); idx >= 0 {
		imageName = imageName[idx+1:]
	}

	// 构建完整的目标镜像引用
	host := registryHost(registryURL)
	var targetRef string
	if namespace != "" {
		targetRef = host + "/" + namespace + "/" + imageName
	} else {
		targetRef = host + "/" + imageName
	}
	if !strings.Contains(targetRef, ":") {
		targetRef += ":latest"
	}

	// 先给镜像打标签
	if err := s.client.ImageTag(ctx, imageRef, targetRef); err != nil {
		logman.Error("Tag image for push failed", "image", imageRef, "target", targetRef, "error", err)
		return "", targetRef, err
	}

	// 推送镜像（认证信息从 targetRef 的 host 自动匹配）
	reader, err := s.client.ImagePush(ctx, targetRef, image.PushOptions{
		RegistryAuth: s.RegistryAuth(targetRef),
	})
	if err != nil {
		logman.Error("Push image failed", "image", targetRef, "error", err)
		return "", targetRef, err
	}
	defer reader.Close()

	lastMessage, err := consumeImageStream(json.NewDecoder(reader))
	if err != nil {
		logman.Error("Push image stream error", "image", targetRef, "error", err)
		return "", targetRef, err
	}

	logman.Info("Image pushed", "image", imageRef, "target", targetRef)
	return lastMessage, targetRef, nil
}

// ImagePull 从仓库拉取镜像到本地
// RegistryURL 为空时直接从 Docker Hub / daemon 配置的 mirror 拉取
func (s *DockerService) ImagePull(ctx context.Context, imageName, registryURL, namespace string) (string, string, error) {
	// 构建完整镜像引用
	var imageRef string
	if registryURL == "" {
		// 无私有仓库：直接使用镜像名，依赖 daemon mirror 配置
		imageRef = imageName
		if !strings.Contains(imageRef, ":") && !strings.Contains(imageRef, "@") {
			imageRef += ":latest"
		}
	} else {
		// 拼接私有仓库完整引用
		host := registryHost(registryURL)
		if namespace != "" {
			imageRef = host + "/" + namespace + "/" + imageName
		} else {
			imageRef = host + "/" + imageName
		}
		if !strings.Contains(imageName, ":") && !strings.Contains(imageName, "@") {
			imageRef += ":latest"
		}
	}

	lastMsg, err := s.imagePull(ctx, imageRef)
	if err != nil {
		return "", imageRef, err
	}

	logman.Info("Image pulled from registry", "image", imageRef, "registry", registryURL)
	return lastMsg, imageRef, nil
}

// imagePull 执行镜像拉取，认证信息从 imageRef 的 host 自动匹配已配置的 registry
func (s *DockerService) imagePull(ctx context.Context, imageRef string) (string, error) {
	reader, err := s.client.ImagePull(ctx, imageRef, image.PullOptions{
		RegistryAuth: s.RegistryAuth(imageRef),
	})
	if err != nil {
		logman.Error("Pull image failed", "image", imageRef, "error", err)
		return "", fmt.Errorf("拉取镜像 %s 失败: %w", imageRef, err)
	}
	defer reader.Close()

	msg, err := consumeImageStream(json.NewDecoder(reader))
	if err != nil {
		logman.Error("Pull image stream error", "image", imageRef, "error", err)
		return "", fmt.Errorf("拉取镜像 %s 失败: %w", imageRef, err)
	}
	return msg, nil
}
