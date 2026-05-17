package registry

import (
	"context"
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/apisix"

	"github.com/rehiy/libgo/logman"
)

var ApisixClient *apisix.Client

// initApisix 初始化 Apisix 服务
func initApisix() error {
	if config.Apisix.AdminURL == "" {
		return fmt.Errorf("apisix adminUrl not configured")
	}

	ApisixClient = apisix.NewClient(config.Apisix.AdminURL, config.Apisix.AdminKey)
	return nil
}

// IsApisixAvailable 检查 Apisix 是否可用（支持 context 超时/取消）
func IsApisixAvailable(ctx context.Context) bool {
	if ApisixClient == nil {
		logman.Warn("Apisix client not initialized")
		return false
	}

	_, err := ApisixClient.RouteList(ctx)
	if err != nil {
		logman.Error("Apisix client not available", "error", err)
		return false
	}
	return true
}
