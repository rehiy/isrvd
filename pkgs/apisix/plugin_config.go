package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// PluginConfig Apisix Plugin Config 信息
type PluginConfig struct {
	ID         string         `json:"id"`
	Desc       string         `json:"desc"`
	Plugins    map[string]any `json:"plugins,omitempty"`
	CreateTime int64          `json:"create_time"`
	UpdateTime int64          `json:"update_time"`
}

// ListPluginConfigs 获取所有 Plugin Config 列表
func (c *Client) ListPluginConfigs() ([]PluginConfig, error) {
	data, err := c.doRequest(http.MethodGet, "/plugin_configs", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Plugin Config 列表失败: %w", err)
	}
	return parsePluginConfigList(data)
}

// parsePluginConfigList 解析 Apisix Plugin Config 列表响应
func parsePluginConfigList(data []byte) ([]PluginConfig, error) {
	var raw struct {
		List []struct {
			Value PluginConfig `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Plugin Config 列表失败: %w", err)
	}

	result := make([]PluginConfig, 0, len(raw.List))
	for _, item := range raw.List {
		result = append(result, item.Value)
	}
	return result, nil
}
