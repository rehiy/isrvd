// Package compose 提供统一的 Compose 部署业务服务
package compose

import (
	"fmt"
	"sync"

	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
)

var (
	once          sync.Once
	deployService *DeployService
)

// NewService 创建 compose 部署服务（供 server 层调用）
func NewService() (*DeployService, error) {
	once.Do(func() {
		d := registry.DockerService
		s := registry.SwarmService
		c := registry.ComposeService

		if d == nil {
			logman.Warn("Compose service: docker service not available")
			return
		}

		if c != nil {
			if dp, err := NewDeployService(d, c, s); err != nil {
				logman.Warn("DeployService init failed", "error", err)
			} else {
				deployService = dp
			}
		} else {
			logman.Warn("Compose service: compose pkg service not available, deploy disabled")
		}
	})

	if deployService == nil {
		return nil, fmt.Errorf("Compose 部署服务未初始化")
	}
	return deployService, nil
}
