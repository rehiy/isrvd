package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rehiy/pango/logman"
)

// ListImages 列出镜像
func (s *DockerService) ListImages(ctx context.Context, all bool) ([]*ImageInfo, error) {
	images, err := s.client.ImageList(ctx, types.ImageListOptions{All: all})
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

// ImageAction 镜像操作
func (s *DockerService) ImageAction(ctx context.Context, id, action string) error {
	switch action {
	case "remove":
		_, err := s.client.ImageRemove(ctx, id, types.ImageRemoveOptions{
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

// PullImage 拉取镜像
func (s *DockerService) PullImage(ctx context.Context, image, tag string) (string, string, error) {
	imageRef := image
	if tag != "" {
		imageRef = image + ":" + tag
	} else if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
		imageRef = image + ":latest"
	}

	reader, err := s.client.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		logman.Error("Pull image failed", "image", imageRef, "error", err)
		return "", imageRef, err
	}
	defer reader.Close()

	var lastMessage string
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
			logman.Error("Pull image stream error", "image", imageRef, "error", msg.Error)
			return "", imageRef, fmt.Errorf(msg.Error)
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pulled", "image", imageRef)
	return lastMessage, imageRef, nil
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

// SearchImages 搜索镜像
func (s *DockerService) SearchImages(ctx context.Context, term string) ([]*ImageSearchResult, error) {
	results, err := s.client.ImageSearch(ctx, term, types.ImageSearchOptions{Limit: 25})
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

	resp, err := s.client.ImageBuild(ctx, tarBuf, types.ImageBuildOptions{
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
			return "", fmt.Errorf(msg.Error)
		}
		if msg.Stream != "" {
			lastMessage = strings.TrimSpace(msg.Stream)
		}
	}

	logman.Info("Image built", "tag", tag)
	return lastMessage, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
