// Package compose 提供统一的 Compose 部署业务服务
package compose

import (
	"fmt"
	"sync"

	"github.com/rehiy/pango/logman"

	"isrvd/internal/registry"
)

var (
	once            sync.Once
	snapshotService *SnapshotService
	deployService   *DeployService
)

// GetSnapshotService 返回全局快照服务实例（供 docker service 使用）
func GetSnapshotService() *SnapshotService {
	initServices()
	return snapshotService
}

// GetDeployService 返回全局部署服务实例
func GetDeployService() *DeployService {
	initServices()
	return deployService
}

// initServices 懒初始化业务服务（只执行一次）
func initServices() {
	once.Do(func() {
		d := registry.DockerService
		c := registry.ComposeService
		s := registry.SwarmManager

		if d == nil {
			logman.Warn("Compose service: docker service not available")
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
		} else {
			logman.Warn("Compose service: compose pkg service not available, deploy disabled")
		}
	})
}

// NewComposeService 创建 compose 服务（供 server 层调用）
func NewComposeService() (*DeployService, error) {
	initServices()
	if deployService == nil {
		return nil, fmt.Errorf("Compose 部署服务未初始化")
	}
	return deployService, nil
}
