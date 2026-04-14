package docker

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"

)

// ListImages 列出镜像
func (h *DockerHandler) ListImages(c *gin.Context) {
	ctx := c.Request.Context()
	all := c.DefaultQuery("all", "false") == "true"

	images, err := h.dockerClient.ImageList(ctx, types.ImageListOptions{All: all})
	if err != nil {
		logman.Error("List images failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取镜像列表失败")
		return
	}

	var result []*ImageInfo
	for _, img := range images {
		// 过滤掉中间层镜像（没有 RepoTags 的镜像）
		if !all && len(img.RepoTags) == 0 {
			continue
		}

		id := img.ID
		// 计算短ID用于显示
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

	helper.RespondSuccess(c, "Images listed successfully", result)
}

// ImageAction 镜像操作
func (h *DockerHandler) ImageAction(c *gin.Context) {
	var req ImageActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Image action failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	switch req.Action {
	case "remove":
		// 直接使用传入的ID（已经是完整的sha256:xxx格式）
		// Force: 强制删除，即使镜像有多个标签引用
		// PruneChildren: 自动删除未被其他镜像引用的子镜像（中间层镜像）
		_, err := h.dockerClient.ImageRemove(ctx, req.ID, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			logman.Error("Remove image failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "删除镜像失败: "+err.Error())
			return
		}
	default:
		logman.Error("Unsupported image action", "action", req.Action)
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Image action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, "Image action performed successfully", nil)
}

// PullImage 拉取镜像
func (h *DockerHandler) PullImage(c *gin.Context) {
	var req ImagePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Pull image failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	imageRef := req.Image
	if req.Tag != "" {
		imageRef = req.Image + ":" + req.Tag
	} else if !strings.Contains(req.Image, ":") && !strings.Contains(req.Image, "@") {
		imageRef = req.Image + ":latest"
	}

	reader, err := h.dockerClient.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		logman.Error("Pull image failed", "image", imageRef, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+err.Error())
		return
	}
	defer reader.Close()

	var lastMessage string
	decoder := json.NewDecoder(reader)
	for {
		var msg struct {
			Status         string `json:"status"`
			Error          string `json:"error"`
			ProgressDetail struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			} `json:"progressDetail"`
		}
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			logman.Error("Pull image stream error", "image", imageRef, "error", msg.Error)
			helper.RespondError(c, http.StatusInternalServerError, "拉取镜像失败: "+msg.Error)
			return
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}

	logman.Info("Image pulled", "image", imageRef)
	helper.RespondSuccess(c, "Image pulled successfully", gin.H{"image": imageRef, "status": lastMessage})
}

// ImageTag 镜像打标签
func (h *DockerHandler) ImageTag(c *gin.Context) {
	var req ImageTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Tag image failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()

	if err := h.dockerClient.ImageTag(ctx, req.ID, req.RepoTag); err != nil {
		logman.Error("Tag image failed", "id", req.ID, "tag", req.RepoTag, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "镜像打标签失败: "+err.Error())
		return
	}

	logman.Info("Image tagged", "id", req.ID, "tag", req.RepoTag)
	helper.RespondSuccess(c, "Image tagged successfully", nil)
}

// ImageSearch 搜索镜像
func (h *DockerHandler) ImageSearch(c *gin.Context) {
	term := c.Query("term")
	if term == "" {
		logman.Error("Search image failed", "error", "search term is empty")
		helper.RespondError(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	ctx := c.Request.Context()

	results, err := h.dockerClient.ImageSearch(ctx, term, types.ImageSearchOptions{Limit: 25})
	if err != nil {
		logman.Error("Search image failed", "term", term, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "搜索镜像失败: "+err.Error())
		return
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

	helper.RespondSuccess(c, "Image search completed", searchResults)
}

// ImageBuild 构建镜像
func (h *DockerHandler) ImageBuild(c *gin.Context) {
	var req ImageBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Build image failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()

	// 构建 Dockerfile 的 tar 包
	tarBuf, err := buildDockerfileTar(req.Dockerfile)
	if err != nil {
		logman.Error("Build dockerfile tar failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "构建 Dockerfile 包失败: "+err.Error())
		return
	}

	tag := req.Tag
	if tag == "" {
		tag = "custom:latest"
	}

	resp, err := h.dockerClient.ImageBuild(ctx, tarBuf, types.ImageBuildOptions{
		Tags: []string{tag},
	})
	if err != nil {
		logman.Error("Build image failed", "tag", tag, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "构建镜像失败: "+err.Error())
		return
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
			helper.RespondError(c, http.StatusInternalServerError, "构建镜像失败: "+msg.Error)
			return
		}
		if msg.Stream != "" {
			lastMessage = strings.TrimSpace(msg.Stream)
		}
	}

	logman.Info("Image built", "tag", tag)
	helper.RespondSuccess(c, "Image built successfully", gin.H{"tag": tag, "status": lastMessage})
}
