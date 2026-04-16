// Package registry 提供服务注册和管理功能
// 统一管理 Docker、Apisix 等外部服务的连接
package registry

import (
	"sync"

	"github.com/docker/docker/client"

	"isrvd/pkgs/apisix"
	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
)

// Registry 服务注册表，管理所有外部服务连接
type Registry struct {
	// Docker 相关服务
	dockerService *docker.DockerService
	swarmManager  *swarm.SwarmManager
	dockerClient  *client.Client

	// Apisix 服务
	apisixClient *apisix.Client

	// 初始化状态
	mu     sync.RWMutex
	inited bool
}

// DefaultRegistry 默认注册表实例
var DefaultRegistry = &Registry{}

// Initialize 初始化所有服务连接
func (r *Registry) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inited {
		return nil
	}

	// 初始化 Docker 服务（可选）
	_ = r.initDocker()

	// 初始化 Apisix 服务（可选）
	_ = r.initApisix()

	r.inited = true
	return nil
}


