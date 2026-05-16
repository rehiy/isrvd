package overview

import (
	"context"
	"time"

	"isrvd/config"
	"isrvd/internal/registry"
)

// ProbeResponse 探活响应
type ProbeResponse struct {
	Agent   map[string]bool `json:"agent"`
	Apisix  map[string]bool `json:"apisix"`
	Caddy   map[string]bool `json:"caddy"`
	Docker  map[string]bool `json:"docker"`
	Swarm   map[string]bool `json:"swarm"`
	Compose map[string]bool `json:"compose"`
}

// UptimeResponse 服务启动时间响应
type UptimeResponse struct {
	StartTime int64 `json:"startTime"`
	Uptime    int64 `json:"uptime"`
}

var startTime = time.Now()

// Uptime 获取服务启动时间
func (s *Service) Uptime() *UptimeResponse {
	return &UptimeResponse{
		StartTime: startTime.Unix(),
		Uptime:    int64(time.Since(startTime).Seconds()),
	}
}

// Probe 服务探活
func (s *Service) Probe(ctx context.Context) *ProbeResponse {
	return &ProbeResponse{
		Agent:   map[string]bool{"available": config.Agent.BaseURL != "" && config.Agent.APIKey != ""},
		Apisix:  map[string]bool{"available": registry.IsApisixAvailable()},
		Caddy:   map[string]bool{"available": registry.IsCaddyAvailable(ctx)},
		Docker:  map[string]bool{"available": registry.IsDockerAvailable(ctx)},
		Swarm:   map[string]bool{"available": registry.IsSwarmAvailable(ctx)},
		Compose: map[string]bool{"available": registry.IsComposeAvailable(ctx)},
	}
}
