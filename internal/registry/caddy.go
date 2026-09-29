package registry

import (
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/caddy"
)

var CaddyClient *caddy.Client

// initCaddy 初始化 Caddy Admin API 客户端
func initCaddy() error {
	CaddyClient = nil
	cfg := config.Current().Caddy
	if cfg.AdminURL == "" {
		return fmt.Errorf("caddy adminUrl not configured")
	}
	CaddyClient = caddy.NewClient(cfg.AdminURL)
	return nil
}
