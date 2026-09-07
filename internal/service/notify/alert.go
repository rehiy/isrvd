package notify

import (
	"fmt"
	"sync"

	"isrvd/config"
)

// 支持的资源指标
const (
	MetricCPU    = "cpu"
	MetricMemory = "memory"
	MetricDisk   = "disk"
)

// ruleState 记录单条规则的连续超阈值次数与当前是否处于告警中，
// 用于抑制瞬时抖动并避免重复推送。
type ruleState struct {
	exceed int
	firing bool
}

var (
	statesMu sync.Mutex
	states   = map[string]*ruleState{}
)

// HostUsage 主机资源使用率，由监控采集数据换算而来
type HostUsage struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
}

// CheckHost 按配置规则检查主机资源使用率，触发或恢复时发送告警
func CheckHost(usage *HostUsage) {
	if config.Notify == nil || len(config.Notify.Rules) == 0 || usage == nil {
		return
	}

	activeKeys := make(map[string]struct{}, len(config.Notify.Rules))
	for _, rule := range config.Notify.Rules {
		if rule != nil {
			activeKeys[alertRuleKey(rule)] = struct{}{}
		}
	}
	statesMu.Lock()
	for key := range states {
		if _, active := activeKeys[key]; !active {
			delete(states, key)
		}
	}
	statesMu.Unlock()

	for _, rule := range config.Notify.Rules {
		if rule == nil {
			continue
		}
		checkRule(alertRuleKey(rule), rule, usage)
	}
}

// checkRule 检查单条规则；状态变更时发送通知
func checkRule(key string, rule *config.AlertRule, usage *HostUsage) {
	value := metricValue(rule.Metric, usage)
	if value < 0 || rule.Threshold <= 0 {
		return
	}

	duration := rule.Duration
	if duration <= 0 {
		duration = 1
	}

	statesMu.Lock()
	st := states[key]
	if st == nil {
		st = &ruleState{}
		states[key] = st
	}

	switch {
	case value >= rule.Threshold:
		st.exceed++
		if st.firing || st.exceed < duration {
			statesMu.Unlock()
			return
		}
		st.firing = true
		statesMu.Unlock()
		Send(buildEvent("resource.alert", rule, value))
	case st.firing:
		// 已回落到阈值以下，发送恢复通知
		st.firing = false
		st.exceed = 0
		statesMu.Unlock()
		Send(buildEvent("resource.recover", rule, value))
	default:
		st.exceed = 0
		statesMu.Unlock()
	}
}

// alertRuleKey 使用规则内容生成状态键，避免依赖易变的数组下标。
func alertRuleKey(rule *config.AlertRule) string {
	return fmt.Sprintf("%s:%.6f:%d", rule.Metric, rule.Threshold, rule.Duration)
}

// buildEvent 构造告警或恢复事件
func buildEvent(event string, rule *config.AlertRule, value float64) *Event {
	label := metricLabel(rule.Metric)
	level := "warning"
	if value >= 90 {
		level = "critical"
	}

	evt := &Event{
		Event:   event,
		Level:   level,
		Title:   fmt.Sprintf("%s %.1f%%，超过阈值 %.1f%%", label, value, rule.Threshold),
		Message: fmt.Sprintf("主机%s达到 %.1f%%，告警阈值 %.1f%%", label, value, rule.Threshold),
		Data: map[string]any{
			"metric":    rule.Metric,
			"value":     value,
			"threshold": rule.Threshold,
			"unit":      "%",
		},
	}
	if event == "resource.recover" {
		evt.Level = "info"
		evt.Title = fmt.Sprintf("%s已恢复至 %.1f%%", label, value)
		evt.Message = fmt.Sprintf("主机%s回落到 %.1f%%，低于告警阈值 %.1f%%", label, value, rule.Threshold)
	}
	return evt
}

// metricValue 取指定指标的当前值，未知指标返回 -1
func metricValue(metric string, usage *HostUsage) float64 {
	switch metric {
	case MetricCPU:
		return usage.CPUPercent
	case MetricMemory:
		return usage.MemoryPercent
	case MetricDisk:
		return usage.DiskPercent
	default:
		return -1
	}
}

// metricLabel 指标中文名
func metricLabel(metric string) string {
	switch metric {
	case MetricCPU:
		return "CPU 使用率"
	case MetricMemory:
		return "内存使用率"
	case MetricDisk:
		return "磁盘使用率"
	default:
		return metric
	}
}
