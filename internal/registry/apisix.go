package registry

import (
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/apisix"
)

var ApisixClient *apisix.Client

// initApisix 初始化 Apisix 服务
func initApisix() error {
	ApisixClient = nil
	cfg := config.Current().Apisix
	if cfg.AdminURL == "" {
		return fmt.Errorf("apisix adminUrl not configured")
	}

	ApisixClient = apisix.NewClient(cfg.AdminURL, cfg.AdminKey)
	return nil
}
