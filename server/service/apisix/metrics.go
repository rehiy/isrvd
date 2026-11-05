package apisix

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"isrvd/server/config"
)

// metricsRuleID 由 isrvd 管理的指标全局规则 ID
const metricsRuleID = "isrvd-metrics"

var ErrMetricsExternallyManaged = errors.New("指标由其他全局规则开启，请在 APISIX 中关闭")

// MetricsURL 返回 APISIX 指标抓取地址：优先使用显式配置；否则取 Admin URL 的主机加默认导出端口 9091。
// 未配置 Admin URL 或无法解析时返回空串。
func MetricsURL(cfg *config.ApisixConfig) string {
	if cfg == nil || cfg.AdminURL == "" {
		return ""
	}
	if cfg.MetricsURL != "" {
		return cfg.MetricsURL
	}
	u, err := url.Parse(cfg.AdminURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return u.Scheme + "://" + net.JoinHostPort(u.Hostname(), "9091") + "/apisix/prometheus/metrics"
}

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
		if !enabled && hasMetrics && rule.ID != metricsRuleID {
			return ErrMetricsExternallyManaged
		}
		if rule.ID == metricsRuleID && rule.Plugins != nil {
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
		if err := s.client.GlobalRuleSave(ctx, metricsRuleID, plugins); err != nil {
			return fmt.Errorf("开启 APISIX 指标失败: %w", err)
		}
		return nil
	}
	if _, ok := plugins["prometheus"]; !ok {
		return nil
	}
	delete(plugins, "prometheus")
	if len(plugins) == 0 {
		err = s.client.GlobalRuleDelete(ctx, metricsRuleID)
	} else {
		err = s.client.GlobalRuleSave(ctx, metricsRuleID, plugins)
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
