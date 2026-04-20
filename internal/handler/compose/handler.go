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

// Deploy 统一的 compose 部署入口
// POST /api/compose/deploy
//
// 行为由 target 决定：
//   - target=docker：落盘到 {ContainerRoot}/{projectName}
//     可选 initURL 指定附加运行文件 zip（应用市场一键安装）
//   - target=swarm ：不落盘，projectName 仅作 compose project 名
func (h *ComposeHandler) Deploy(c *gin.Context) {
	var req composesvc.DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	if registry.ComposeDeployService == nil {
		helper.RespondError(c, http.StatusServiceUnavailable, "Compose 部署服务未初始化")
		return
	}

	result, err := registry.ComposeDeployService.Deploy(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "部署成功", result)
}
