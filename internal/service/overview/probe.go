package overview

import (
	"context"
	"sync"
	"time"

	"isrvd/config"
)

// ProbeResponse 探活响应
type ProbeResponse struct {
	Agent   bool `json:"agent"`   // Agent 服务是否可用
	Apisix  bool `json:"apisix"`  // Apisix 网关是否可用
	Caddy   bool `json:"caddy"`   // Caddy 服务是否可用
	Docker  bool `json:"docker"`  // Docker 引擎是否可用
	Swarm   bool `json:"swarm"`   // Docker Swarm 是否可用
	Compose bool `json:"compose"` // Compose 服务是否可用
}

// probeTask 定义一项探活任务
type probeTask struct {
	name string
	fn   func(context.Context) bool
}

// Probe 服务探活（并发检查，整体 5 秒超时）
// probes 由调用方注入各服务的可用性检查函数，解耦对 registry 包的直接依赖
func (s *Service) Probe(ctx context.Context, probes map[string]func(context.Context) bool) *ProbeResponse {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tasks := make([]probeTask, 0, len(probes))
	for name, fn := range probes {
		tasks = append(tasks, probeTask{name: name, fn: fn})
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
