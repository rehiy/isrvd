package registry

import (
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/apisix"
)

// initApisix 初始化 Apisix 服务
func (r *Registry) initApisix() error {
	if config.Apisix.AdminURL == "" {
		return fmt.Errorf("apisix adminUrl not configured")
	}

	r.apisixClient = apisix.NewClient(config.Apisix.AdminURL, config.Apisix.AdminKey)
	return nil
}

// GetApisix 获取 Apisix 客户端实例
func (r *Registry) GetApisix() *apisix.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.apisixClient
}

// IsApisixAvailable 检查 Apisix 是否可用
func (r *Registry) IsApisixAvailable() bool {
	r.mu.RLock()
	client := r.apisixClient
	r.mu.RUnlock()

	if client == nil {
		return false
	}
	_, err := client.ListRoutes()
	return err == nil
}


