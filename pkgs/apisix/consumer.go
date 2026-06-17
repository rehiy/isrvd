package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rehiy/libgo/strutil"
)

// sensitiveFields 各 auth 插件中需要脱敏/还原的敏感字段名
var sensitiveFields = []string{
	"key",        // key-auth + SSL 证书私钥
	"username",   // basic-auth
	"password",   // basic-auth
	"secret",     // jwt-auth
	"public_key", // jwt-auth
	"key_id",     // hmac-auth
	"secret_key", // hmac-auth
}

// Consumer Apisix Consumer 信息
type Consumer struct {
	Username   string         `json:"username"`          // Consumer 用户名（唯一标识）
	Desc       string         `json:"desc"`              // Consumer 描述
	Plugins    map[string]any `json:"plugins,omitempty"` // 插件配置（如 key-auth、jwt-auth 等认证插件）
	CreateTime int64          `json:"create_time"`       // 创建时间（Unix 时间戳，只读）
	UpdateTime int64          `json:"update_time"`       // 更新时间（Unix 时间戳，只读）
}

// ConsumerList 获取所有 Consumer 列表
func (c *Client) ConsumerList(ctx context.Context) ([]Consumer, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/consumers", nil)
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

// ConsumerRaw 获取指定 Consumer 的完整（未脱敏）数据
func (c *Client) ConsumerRaw(ctx context.Context, username string) (*Consumer, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/consumers/"+url.PathEscape(username), nil)
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

// ConsumerUpdate 更新 Consumer，支持传入完整 plugins，
// 对于脱敏字段（含 ****** 的 key/password）自动替换为原始值
func (c *Client) ConsumerUpdate(ctx context.Context, username, desc string, plugins map[string]any) error {
	raw, err := c.ConsumerRaw(ctx, username)
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
	_, err = c.doRequest(ctx, http.MethodPut, "/consumers/"+url.PathEscape(username), body)
	return err
}

// ConsumerCreate 创建 Consumer，支持传入完整 plugins。
// 若 plugins 为空，自动生成 key-auth 插件并返回明文 API Key。
func (c *Client) ConsumerCreate(ctx context.Context, username, desc string, plugins map[string]any) (*Consumer, error) {
	if len(plugins) == 0 {
		apiKey := strutil.Rand(32)
		plugins = map[string]any{
			"key-auth": map[string]any{"key": apiKey},
		}
	}
	body := map[string]any{
		"username": username,
		"desc":     desc,
		"plugins":  plugins,
	}
	_, err := c.doRequest(ctx, http.MethodPut, "/consumers/"+url.PathEscape(username), body)
	if err != nil {
		return nil, err
	}
	maskConsumerPlugins(plugins)
	return &Consumer{Username: username, Desc: desc, Plugins: plugins}, nil
}

// ConsumerDelete 删除指定 Consumer
func (c *Client) ConsumerDelete(ctx context.Context, username string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/consumers/"+url.PathEscape(username), nil)
	return err
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
		for _, field := range sensitiveFields {
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

// maskConsumerPlugins 对 plugins 中的敏感字段（key、password 等）进行脱敏：
// - 长度 > 10：保留前 5 位 + ****** + 后 3 位
// - 长度 <= 10：保留首尾各 1 位，中间替换为 ******
// 按 rune 处理，正确支持 UTF-8 多字节字符。
func maskConsumerPlugins(plugins map[string]any) {
	for _, plugin := range plugins {
		p, ok := plugin.(map[string]any)
		if !ok {
			continue
		}
		for _, field := range sensitiveFields {
			s, ok := p[field].(string)
			if !ok || len(s) == 0 {
				continue
			}
			runes := []rune(s)
			n := len(runes)
			if n > 10 {
				p[field] = string(runes[:5]) + "******" + string(runes[n-3:])
			} else if n > 2 {
				p[field] = string(runes[:1]) + "******" + string(runes[n-1:])
			} else {
				p[field] = "******"
			}
		}
	}
}
