package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ListPlugins 获取 APISIX 中所有可用 Plugin 及其 Schema
func (c *Client) ListPlugins() (map[string]any, error) {
	data, err := c.doRequest(http.MethodGet, "/plugins?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析插件列表失败: %w", err)
	}
	return result, nil
}

// --- 辅助函数 ---

// hasPlugin 检查插件列表中是否存在指定插件
func hasPlugin(plugins map[string]any, name string) bool {
	if plugins == nil {
		return false
	}
	_, ok := plugins[name]
	return ok
}

// pluginConsumerRestrictionWhitelist 从 plugins 中提取 consumer-restriction 白名单
func pluginConsumerRestrictionWhitelist(plugins map[string]any) []string {
	if plugins == nil {
		return []string{}
	}

	cr, ok := plugins["consumer-restriction"].(map[string]any)
	if !ok {
		return []string{}
	}

	wl, ok := cr["whitelist"].([]any)
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(wl))
	for _, v := range wl {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
