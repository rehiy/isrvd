// Package compose 提供统一的 Compose 部署 HTTP 入口
package compose

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/internal/helper"
	"isrvd/internal/registry"
)

var (
	once            sync.Once
	snapshotService *SnapshotService
	deployService   *DeployService
)

// GetSnapshotService 返回全局快照服务实例（供 docker handler 使用）
func GetSnapshotService() *SnapshotService {
	initServices()
	return snapshotService
}

// ComposeHandler Compose 部署处理器
type ComposeHandler struct {
	service *DeployService
}

// NewComposeHandler 创建 Compose 处理器
func NewComposeHandler() (*ComposeHandler, error) {
	initServices()
	if deployService == nil {
		return nil, fmt.Errorf("Compose 部署服务未初始化")
	}
	return &ComposeHandler{service: deployService}, nil
}

// Deploy 统一的 compose 部署入口
func (h *ComposeHandler) Deploy(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	result, err := h.service.Deploy(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "部署失败: "+err.Error())
		return
	}

	helper.RespondSuccess(c, "部署成功", result)
}

// initServices 懒初始化业务服务（只执行一次）
func initServices() {
	once.Do(func() {
		d := registry.DockerService
		c := registry.ComposeService
		s := registry.SwarmManager

		if d == nil {
			logman.Warn("Compose handler: docker service not available")
			return
		}

		if snap, err := NewSnapshotService(d); err != nil {
			logman.Warn("Snapshot service init failed", "error", err)
		} else {
			snapshotService = snap
		}

		if c != nil {
			if dp, err := NewDeployService(d, c, s); err != nil {
				logman.Warn("DeployService init failed", "error", err)
			} else {
				deployService = dp
			}
		}
	})
}
