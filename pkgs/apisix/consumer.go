package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rehiy/pango/strutil"
)

// Consumer Apisix Consumer 信息
type Consumer struct {
	Username   string         `json:"username"`
	Desc       string         `json:"desc"`
	Plugins    map[string]any `json:"plugins,omitempty"`
	CreateTime int64          `json:"create_time"`
	UpdateTime int64          `json:"update_time"`
}

// ListConsumers 获取所有 Consumer 列表
func (c *Client) ListConsumers() ([]Consumer, error) {
	data, err := c.doRequest(http.MethodGet, "/consumers", nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		List []struct {
			Value Consumer `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Consumer 列表失败: %w", err)
	}

	result := make([]Consumer, 0, len(raw.List))
	for _, item := range raw.List {
		maskConsumerPlugins(item.Value.Plugins)
		result = append(result, item.Value)
	}
	return result, nil
}

// GetConsumerRawKey 获取指定 Consumer 的完整（未脱敏）API Key
func (c *Client) GetConsumerRawKey(username string) (string, error) {
	raw, err := c.GetConsumerRaw(username)
	if err != nil {
		return "", err
	}
	if raw.Plugins != nil {
		if keyAuth, ok := raw.Plugins["key-auth"].(map[string]any); ok {
			if key, ok := keyAuth["key"].(string); ok && key != "" {
				return key, nil
			}
		}
	}
	return "", fmt.Errorf("该用户尚未配置 API Key")
}

// GetConsumerRaw 获取指定 Consumer 的完整（未脱敏）数据
func (c *Client) GetConsumerRaw(username string) (*Consumer, error) {
	data, err := c.doRequest(http.MethodGet, "/consumers/"+username, nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Value Consumer `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Consumer 失败: %w", err)
	}
	return &raw.Value, nil
}

// UpdateConsumer 更新 Consumer，支持传入完整 plugins，
// 对于脱敏字段（含 ****** 的 key/password）自动替换为原始值
func (c *Client) UpdateConsumer(username, desc string, plugins map[string]any) error {
	raw, err := c.GetConsumerRaw(username)
	if err != nil {
		return err
	}

	// 用原始值替换脱敏字段
	if raw.Plugins != nil && plugins != nil {
		unmaskPlugins(plugins, raw.Plugins)
	}

	body := map[string]any{
		"username": username,
		"desc":     desc,
		"plugins":  plugins,
	}
	_, err = c.doRequest(http.MethodPut, "/consumers/"+username, body)
	return err
}

// CreateConsumer 创建 Consumer，自动生成 API Key，返回完整的 Consumer（含明文 API Key）
func (c *Client) CreateConsumer(username, desc string) (*Consumer, error) {
	apiKey := strutil.Rand(32)
	if err := c.CreateOrUpdateConsumer(username, apiKey, desc); err != nil {
		return nil, err
	}
	return &Consumer{
		Username: username,
		Desc:     desc,
		Plugins: map[string]any{
			"key-auth": map[string]any{
				"key": apiKey,
			},
		},
	}, nil
}

// UpdateConsumerDesc 更新 Consumer 描述，保持原有 API Key 不变
func (c *Client) UpdateConsumerDesc(username, desc string) error {
	apiKey, err := c.GetConsumerRawKey(username)
	if err != nil {
		return err
	}
	return c.CreateOrUpdateConsumer(username, apiKey, desc)
}

// CreateOrUpdateConsumer 创建或更新 Consumer（key-auth 插件）
func (c *Client) CreateOrUpdateConsumer(username, apiKey, desc string) error {
	body := map[string]any{
		"username": username,
		"desc":     desc,
		"plugins": map[string]any{
			"key-auth": map[string]any{
				"key": apiKey,
			},
		},
	}
	_, err := c.doRequest(http.MethodPut, "/consumers/"+username, body)
	if err != nil {
		return err
	}
	return nil
}

// DeleteConsumer 删除指定 Consumer
func (c *Client) DeleteConsumer(username string) error {
	_, err := c.doRequest(http.MethodDelete, "/consumers/"+username, nil)
	if err != nil {
		return err
	}
	return nil
}

// --- 辅助函数 ---

// unmaskPlugins 将 plugins 中的脱敏值（包含 ******）替换为原始值
func unmaskPlugins(plugins, rawPlugins map[string]any) {
	for pluginName, plugin := range plugins {
		p, ok := plugin.(map[string]any)
		if !ok {
			continue
		}
		rawPlugin, ok := rawPlugins[pluginName]
		if !ok {
			continue
		}
		rawP, ok := rawPlugin.(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"key", "password"} {
			val, ok := p[field].(string)
			if !ok || !strings.Contains(val, "******") {
				continue
			}
			if rawVal, ok := rawP[field].(string); ok && rawVal != "" {
				p[field] = rawVal
			}
		}
	}
}

// maskConsumerPlugins 对 plugins 中的敏感字段（key、password）进行脱敏：
// - 长度 > 10：保留前 5 位 + ****** + 后 3 位
// - 长度 <= 10：保留首尾各 1 位，中间替换为 ******
func maskConsumerPlugins(plugins map[string]any) {
	for _, plugin := range plugins {
		p, ok := plugin.(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"key", "password"} {
			s, ok := p[field].(string)
			if !ok || len(s) == 0 {
				continue
			}
			n := len(s)
			if n > 10 {
				p[field] = s[:5] + "******" + s[n-3:]
			} else if n > 2 {
				p[field] = s[:1] + "******" + s[n-1:]
			} else {
				p[field] = "******"
			}
		}
	}
}
