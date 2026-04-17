package docker

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/config"
	"isrvd/internal/helper"
	"isrvd/pkgs/docker"
)

// ListRegistries 列出已配置的镜像仓库
func (h *DockerHandler) ListRegistries(c *gin.Context) {
	result := h.service.ListRegistries()
	helper.RespondSuccess(c, "Registries listed successfully", result)
}

// RegistryUpsertRequest 仓库新建/更新请求
type RegistryUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

// syncRegistriesToConfig 将当前 DockerService 的仓库同步到全局 config 并落盘
func (h *DockerHandler) syncRegistriesToConfig() error {
	regs := h.service.GetRegistryConfigs()
	cfgRegs := make([]*config.DockerRegistry, 0, len(regs))
	for _, r := range regs {
		cfgRegs = append(cfgRegs, &config.DockerRegistry{
			Name:        r.Name,
			Description: r.Description,
			URL:         r.URL,
			Username:    r.Username,
			Password:    r.Password,
		})
	}
	config.Docker.Registries = cfgRegs
	return config.Save()
}

// CreateRegistry 新建镜像仓库
func (h *DockerHandler) CreateRegistry(c *gin.Context) {
	var req RegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	reg := &docker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	if err := h.service.AddRegistry(reg); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.syncRegistriesToConfig(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "仓库添加成功", nil)
}

// UpdateRegistry 更新镜像仓库
func (h *DockerHandler) UpdateRegistry(c *gin.Context) {
	originalURL := c.Query("url")
	if originalURL == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	var req RegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	reg := &docker.RegistryConfig{
		Name:        req.Name,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
	}
	if err := h.service.UpdateRegistry(originalURL, reg); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.syncRegistriesToConfig(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "仓库更新成功", nil)
}

// DeleteRegistry 删除镜像仓库
func (h *DockerHandler) DeleteRegistry(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		helper.RespondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	if err := h.service.DeleteRegistry(url); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.syncRegistriesToConfig(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	helper.RespondSuccess(c, "仓库删除成功", nil)
}

// PushImage 推送镜像到仓库
func (h *DockerHandler) PushImage(c *gin.Context) {
	var req docker.ImagePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, targetRef, err := h.service.PushImage(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "推送镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像推送成功", gin.H{"image": req.Image, "target": targetRef, "message": msg})
}

// PullFromRegistry 从仓库拉取镜像
func (h *DockerHandler) PullFromRegistry(c *gin.Context) {
	var req docker.ImagePullFromRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	msg, imageRef, err := h.service.PullFromRegistry(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "从仓库拉取镜像失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "镜像拉取成功", gin.H{"image": imageRef, "message": msg})
}
