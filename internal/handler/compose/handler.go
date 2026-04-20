// Package compose 提供统一的 Compose 部署 HTTP 入口
package compose

import (
	"net/http"

	"github.com/gin-gonic/gin"

	composesvc "isrvd/internal/service/compose"

	"isrvd/internal/helper"
	"isrvd/internal/registry"
)

// ComposeHandler Compose 部署处理器
type ComposeHandler struct{}

// NewComposeHandler 创建 Compose 处理器
func NewComposeHandler() *ComposeHandler {
	return &ComposeHandler{}
}

// DeployYml 基于 compose 文本的部署入口
// POST /api/compose/deploy/yml
//
// 根据 target 字段分发：
//   - target=docker：将 yaml 文本部署为单机容器
//   - target=swarm：将 yaml 文本部署为 swarm 服务
func (h *ComposeHandler) DeployYml(c *gin.Context) {
	var req composesvc.DeployYmlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	if registry.ComposeDeployService == nil {
		helper.RespondError(c, http.StatusServiceUnavailable, "Compose 部署服务未初始化")
		return
	}

	result, err := registry.ComposeDeployService.DeployYml(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "部署成功", result)
}

// DeployZip 基于 zip 压缩包的部署入口（仅 docker）
// POST /api/compose/deploy/zip
//
// 下载 zip → 解压到 {containerRoot}/{name} → 写 .env → 部署为单机容器
func (h *ComposeHandler) DeployZip(c *gin.Context) {
	var req composesvc.DeployZipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	if registry.ComposeDeployService == nil {
		helper.RespondError(c, http.StatusServiceUnavailable, "Compose 部署服务未初始化")
		return
	}

	result, err := registry.ComposeDeployService.DeployZip(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "部署成功", result)
}
