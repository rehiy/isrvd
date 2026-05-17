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
	Agent   bool `json:"agent"`
	Apisix  bool `json:"apisix"`
	Caddy   bool `json:"caddy"`
	Docker  bool `json:"docker"`
	Swarm   bool `json:"swarm"`
	Compose bool `json:"compose"`
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

// probeTask 定义一项探活任务
type probeTask struct {
	name string
	fn   func(context.Context) bool
}

// Probe 服务探活（并发检查，整体 5 秒超时）
func (s *Service) Probe(ctx context.Context) *ProbeResponse {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tasks := []probeTask{
		{name: "Apisix", fn: registry.IsApisixAvailable},
		{name: "Caddy", fn: registry.IsCaddyAvailable},
		{name: "Docker", fn: registry.IsDockerAvailable},
		{name: "Swarm", fn: registry.IsSwarmAvailable},
		{name: "Compose", fn: registry.IsComposeAvailable},
	}

	resp := &ProbeResponse{
		Agent: config.Agent.BaseURL != "" && config.Agent.APIKey != "",
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	wg.Add(len(tasks))
	for _, t := range tasks {
		go func(t probeTask) {
			defer wg.Done()
			ok := t.fn(probeCtx)
			mu.Lock()
			switch t.name {
			case "Apisix":
				resp.Apisix = ok
			case "Caddy":
				resp.Caddy = ok
			case "Docker":
				resp.Docker = ok
			case "Swarm":
				resp.Swarm = ok
			case "Compose":
				resp.Compose = ok
			}
			mu.Unlock()
		}(t)
	}
	wg.Wait()

	return resp
}
