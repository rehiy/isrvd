package registry

import (
	"context"
	"fmt"

	"isrvd/config"
	"isrvd/pkgs/caddy"

	"github.com/rehiy/libgo/logman"
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

// IsCaddyAvailable 检查 Caddy 是否可用
func IsCaddyAvailable(ctx context.Context) bool {
	if CaddyClient == nil {
		logman.Warn("Caddy client not initialized")
		return false
	}
	if _, err := CaddyClient.ConfigAll(ctx); err != nil {
		logman.Error("Caddy admin not available", "error", err)
		return false
	}
	return true
}
