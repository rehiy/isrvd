package registry

import (
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/apisix"
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
