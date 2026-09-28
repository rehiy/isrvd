package apisix

import (
	"context"
	"net/http"
	"net/url"
)

// PluginConfig Apisix Plugin Config 信息
type PluginConfig struct {
	ID         string         `json:"id,omitempty"`      // PluginConfig ID（创建时指定）
	Desc       string         `json:"desc,omitempty"`    // PluginConfig 描述
	Plugins    map[string]any `json:"plugins,omitempty"` // 插件配置（可被路由引用）
	CreateTime int64          `json:"create_time"`       // 创建时间（Unix 时间戳，只读）
	UpdateTime int64          `json:"update_time"`       // 更新时间（Unix 时间戳，只读）
}

// PluginConfigList 获取所有 Plugin Config 列表
func (c *Client) PluginConfigList(ctx context.Context) ([]PluginConfig, error) {
	return requestResourceList[PluginConfig](ctx, c, "/plugin_configs", "解析 Plugin Config 列表失败")
}

// PluginConfigInspect 获取单个 Plugin Config 详情
func (c *Client) PluginConfigInspect(ctx context.Context, configID string) (*PluginConfig, error) {
	return requestResource[PluginConfig](ctx, c, http.MethodGet, "/plugin_configs/"+url.PathEscape(configID), nil, "解析 Plugin Config 详情失败")
}

// PluginConfigCreate 创建 Plugin Config
func (c *Client) PluginConfigCreate(ctx context.Context, req PluginConfig) (*PluginConfig, error) {
	return requestResource[PluginConfig](ctx, c, http.MethodPut, "/plugin_configs/"+url.PathEscape(req.ID), buildPluginConfigBody(req), "解析 Plugin Config 详情失败")
}

// PluginConfigUpdate 更新 Plugin Config
func (c *Client) PluginConfigUpdate(ctx context.Context, configID string, req PluginConfig) (*PluginConfig, error) {
	return requestResource[PluginConfig](ctx, c, http.MethodPut, "/plugin_configs/"+url.PathEscape(configID), buildPluginConfigBody(req), "解析 Plugin Config 详情失败")
}

// PluginConfigDelete 删除 Plugin Config
func (c *Client) PluginConfigDelete(ctx context.Context, configID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/plugin_configs/"+url.PathEscape(configID), nil)
	return err
}

// ─── 辅助函数 ───

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
