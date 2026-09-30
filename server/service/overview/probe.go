package overview

import (
	"context"
	"sync"
	"time"

	"isrvd/server/config"
)

// ProbeResponse 探活响应
type ProbeResponse struct {
	Copilot bool `json:"copilot"` // Copilot 服务是否可用
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
// probes 由调用方注入各模块的可用性检查函数（key 为模块名，nil 表示无需探活）
func (s *Service) Probe(ctx context.Context, probes map[string]func(context.Context) bool) *ProbeResponse {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tasks := make([]probeTask, 0, len(probes))
	for name, fn := range probes {
		if fn != nil {
			tasks = append(tasks, probeTask{name: name, fn: fn})
		}
	}

	copilot := config.Current().Copilot
	resp := &ProbeResponse{
		Copilot: copilot.BaseURL != "" && copilot.APIKey != "",
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
			case "apisix":
				resp.Apisix = ok
			case "caddy":
				resp.Caddy = ok
			case "docker":
				resp.Docker = ok
			case "swarm":
				resp.Swarm = ok
			case "compose":
				resp.Compose = ok
			}
			mu.Unlock()
		}(t)
	}
	wg.Wait()

	return resp
}
