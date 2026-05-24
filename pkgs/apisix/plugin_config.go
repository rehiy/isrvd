package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// PluginConfig Apisix Plugin Config 信息
type PluginConfig struct {
	ID         string         `json:"id,omitempty"`
	Desc       string         `json:"desc,omitempty"`
	Plugins    map[string]any `json:"plugins,omitempty"`
	CreateTime int64          `json:"create_time"`
	UpdateTime int64          `json:"update_time"`
}

// PluginConfigList 获取所有 Plugin Config 列表
func (c *Client) PluginConfigList(ctx context.Context) ([]PluginConfig, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/plugin_configs", nil)
	if err != nil {
		return nil, err
	}
	return parsePluginConfigList(data)
}

// PluginConfigInspect 获取单个 Plugin Config 详情
func (c *Client) PluginConfigInspect(ctx context.Context, configID string) (*PluginConfig, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/plugin_configs/"+url.PathEscape(configID), nil)
	if err != nil {
		return nil, err
	}
	return parseSinglePluginConfig(data)
}

// PluginConfigCreate 创建 Plugin Config
func (c *Client) PluginConfigCreate(ctx context.Context, req PluginConfig) (*PluginConfig, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/plugin_configs/"+url.PathEscape(req.ID), buildPluginConfigBody(req))
	if err != nil {
		return nil, err
	}
	return parseSinglePluginConfig(data)
}

// PluginConfigUpdate 更新 Plugin Config
func (c *Client) PluginConfigUpdate(ctx context.Context, configID string, req PluginConfig) (*PluginConfig, error) {
	data, err := c.doRequest(ctx, http.MethodPatch, "/plugin_configs/"+url.PathEscape(configID), buildPluginConfigBody(req))
	if err != nil {
		return nil, err
	}
	return parseSinglePluginConfig(data)
}

// PluginConfigDelete 删除 Plugin Config
func (c *Client) PluginConfigDelete(ctx context.Context, configID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/plugin_configs/"+url.PathEscape(configID), nil)
	return err
}

// --- 辅助函数 ---

// buildPluginConfigBody 将 Plugin Config 转换为 Apisix API 请求体
func buildPluginConfigBody(req PluginConfig) map[string]any {
	body := make(map[string]any)
	if req.Desc != "" {
		body["desc"] = req.Desc
	}
	if req.Plugins != nil {
		body["plugins"] = req.Plugins
	} else {
		body["plugins"] = map[string]any{}
	}
	return body
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

// parseSinglePluginConfig 解析单个 Plugin Config 响应
func parseSinglePluginConfig(data []byte) (*PluginConfig, error) {
	var raw struct {
		Value PluginConfig `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Plugin Config 详情失败: %w", err)
	}
	return &raw.Value, nil
}
