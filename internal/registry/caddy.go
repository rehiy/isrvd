package registry

import (
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/caddy"
)

var CaddyClient *caddy.Client

// initCaddy 初始化 Caddy Admin API 客户端
func initCaddy() error {
	if config.Caddy.AdminURL == "" {
		return fmt.Errorf("caddy adminUrl not configured")
	}
	CaddyClient = caddy.NewClient(config.Caddy.AdminURL)
	return nil
}
