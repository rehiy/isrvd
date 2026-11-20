package node

import (
	"context"
	"time"
)

// AlertWatch 只观察中控已激活节点的隧道在线状态；告警启用判定和通知发送均由调用方注入。
// 进程重启或告警关闭会重建观察基线；节点吊销、删除不会触发虚假的恢复通知。
func (s *Service) AlertWatch(ctx context.Context, enabled func() bool, emit func(id, name string, recovered bool, offline time.Duration)) {
	if enabled == nil || emit == nil {
		return
	}
	const interval = 30 * time.Second
	const offlineDelay = 90 * time.Second
	type alertState struct {
		since  time.Time
		firing bool
	}
	states := make(map[string]*alertState)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		if !enabled() {
			clear(states)
		} else {
			now := time.Now()
			seen := make(map[string]bool)
			for _, n := range s.NodeList() {
				if n.Status != StatusActive {
					continue
				}
				seen[n.ID] = true
				state := states[n.ID]
				if n.Online {
					if state != nil && state.firing {
						emit(n.ID, n.Name, true, now.Sub(state.since))
					}
					delete(states, n.ID)
					continue
				}
				if state == nil {
					states[n.ID] = &alertState{since: now}
				} else if !state.firing && now.Sub(state.since) >= offlineDelay {
					state.firing = true
					emit(n.ID, n.Name, false, now.Sub(state.since))
				}
			}
			for id := range states {
				if !seen[id] {
					delete(states, id)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
