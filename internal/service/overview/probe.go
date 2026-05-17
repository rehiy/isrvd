package overview

import (
	"context"
	"sync"
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

// Probe 服务探活（并发检查，整体 5 秒超时）
func (s *Service) Probe(ctx context.Context) *ProbeResponse {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		agentOk, apisixOk, caddyOk, dockerOk, swarmOk, composeOk bool
		wg                                                                 sync.WaitGroup
		mu                                                                 sync.Mutex
	)

	wg.Add(5) // Agent 不需要网络检查，直接判断

	// Agent 检查（本地配置判断，无需并发）
	agentOk = config.Agent.BaseURL != "" && config.Agent.APIKey != ""

	// Apisix 检查
	go func() {
		defer wg.Done()
		ok := registry.IsApisixAvailable(probeCtx)
		mu.Lock()
		apisixOk = ok
		mu.Unlock()
	}()

	// Caddy 检查
	go func() {
		defer wg.Done()
		ok := registry.IsCaddyAvailable(probeCtx)
		mu.Lock()
		caddyOk = ok
		mu.Unlock()
	}()

	// Docker 检查
	go func() {
		defer wg.Done()
		ok := registry.IsDockerAvailable(probeCtx)
		mu.Lock()
		dockerOk = ok
		mu.Unlock()
	}()

	// Swarm 检查
	go func() {
		defer wg.Done()
		ok := registry.IsSwarmAvailable(probeCtx)
		mu.Lock()
		swarmOk = ok
		mu.Unlock()
	}()

	// Compose 检查
	go func() {
		defer wg.Done()
		ok := registry.IsComposeAvailable(probeCtx)
		mu.Lock()
		composeOk = ok
		mu.Unlock()
	}()

	wg.Wait()

	return &ProbeResponse{
		Agent:   map[string]bool{"available": agentOk},
		Apisix:  map[string]bool{"available": apisixOk},
		Caddy:   map[string]bool{"available": caddyOk},
		Docker:  map[string]bool{"available": dockerOk},
		Swarm:   map[string]bool{"available": swarmOk},
		Compose: map[string]bool{"available": composeOk},
	}
}
