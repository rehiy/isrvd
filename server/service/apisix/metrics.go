package apisix

import (
	"context"
	"errors"
	"fmt"
)

// MetricsRuleID 由 isrvd 管理的指标全局规则 ID
const MetricsRuleID = "isrvd-metrics"

var ErrMetricsExternallyManaged = errors.New("指标由其他全局规则开启，请在 APISIX 中关闭")

// MetricsStatus 返回 prometheus 插件是否已通过任意 Global Rule 对全部路由启用
func (s *Service) MetricsStatus(ctx context.Context) (bool, error) {
	rules, err := s.client.GlobalRuleList(ctx)
	if err != nil {
		return false, fmt.Errorf("获取全局规则列表失败: %w", err)
	}
	for _, rule := range rules {
		if metricsPluginEnabled(rule.Plugins) {
			return true, nil
		}
	}
	return false, nil
}

// MetricsUpdate 开启或关闭网关指标（APISIX 的 prometheus 插件）。
// 开启时已有全局规则启用该插件则不重复创建；关闭时只移除自管规则中的该插件。
// 指标由其他规则开启时拒绝关闭，不修改用户规则。
func (s *Service) MetricsUpdate(ctx context.Context, enabled bool) error {
	rules, err := s.client.GlobalRuleList(ctx)
	if err != nil {
		return fmt.Errorf("获取全局规则列表失败: %w", err)
	}
	plugins := map[string]any{}
	for _, rule := range rules {
		hasMetrics := metricsPluginEnabled(rule.Plugins)
		if enabled && hasMetrics {
			return nil
		}
		if !enabled && hasMetrics && rule.ID != MetricsRuleID {
			return ErrMetricsExternallyManaged
		}
		if rule.ID == MetricsRuleID && rule.Plugins != nil {
			plugins = rule.Plugins
		}
	}
	if enabled {
		plugin, ok := plugins["prometheus"].(map[string]any)
		if !ok {
			plugin = map[string]any{"prefer_name": true}
		}
		if meta, ok := plugin["_meta"].(map[string]any); ok {
			delete(meta, "disable")
			if len(meta) == 0 {
				delete(plugin, "_meta")
			}
		}
		plugins["prometheus"] = plugin
		if err := s.client.GlobalRuleSave(ctx, MetricsRuleID, plugins); err != nil {
			return fmt.Errorf("开启 APISIX 指标失败: %w", err)
		}
		return nil
	}
	if _, ok := plugins["prometheus"]; !ok {
		return nil
	}
	delete(plugins, "prometheus")
	if len(plugins) == 0 {
		err = s.client.GlobalRuleDelete(ctx, MetricsRuleID)
	} else {
		err = s.client.GlobalRuleSave(ctx, MetricsRuleID, plugins)
	}
	if err != nil {
		return fmt.Errorf("关闭 APISIX 指标失败: %w", err)
	}
	return nil
}

// ─── 辅助函数 ───

func metricsPluginEnabled(plugins map[string]any) bool {
	plugin, exists := plugins["prometheus"]
	if !exists {
		return false
	}
	config, _ := plugin.(map[string]any)
	meta, _ := config["_meta"].(map[string]any)
	disabled, _ := meta["disable"].(bool)
	return !disabled
}
