package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Upstream Apisix Upstream 信息
type Upstream struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name"`
	Desc         string         `json:"desc,omitempty"`
	Type         string         `json:"type"`
	Nodes        any            `json:"nodes,omitempty"`
	HashOn       string         `json:"hash_on,omitempty"`
	Key          string         `json:"key,omitempty"`
	Scheme       string         `json:"scheme,omitempty"`
	PassHost     string         `json:"pass_host,omitempty"`
	UpstreamHost string         `json:"upstream_host,omitempty"`
	Retries      int            `json:"retries,omitempty"`
	RetryTimeout int            `json:"retry_timeout,omitempty"`
	Timeout      map[string]any `json:"timeout,omitempty"`
	CreateTime   int64          `json:"create_time"`
	UpdateTime   int64          `json:"update_time"`
}

// UpstreamList 获取所有 Upstream 列表
func (c *Client) UpstreamList(ctx context.Context) ([]Upstream, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/upstreams", nil)
	if err != nil {
		return nil, err
	}
	return parseUpstreamList(data)
}

// UpstreamInspect 获取单条 Upstream 详情
func (c *Client) UpstreamInspect(ctx context.Context, upstreamID string) (*Upstream, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/upstreams/"+url.PathEscape(upstreamID), nil)
	if err != nil {
		return nil, err
	}
	return parseSingleUpstream(data)
}

// UpstreamCreate 创建 Upstream
func (c *Client) UpstreamCreate(ctx context.Context, req Upstream) (*Upstream, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/upstreams", buildUpstreamBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleUpstream(data)
}

// UpstreamUpdate 更新 Upstream
func (c *Client) UpstreamUpdate(ctx context.Context, upstreamID string, req Upstream) (*Upstream, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/upstreams/"+url.PathEscape(upstreamID), buildUpstreamBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleUpstream(data)
}

// UpstreamDelete 删除 Upstream
func (c *Client) UpstreamDelete(ctx context.Context, upstreamID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/upstreams/"+url.PathEscape(upstreamID), nil)
	return err
}

// --- 辅助函数 ---

// buildUpstreamBody 将 Upstream 转换为 Apisix API 请求体
func buildUpstreamBody(req Upstream) map[string]any {
	body := map[string]any{
		"name":  req.Name,
		"type":  req.Type,
		"nodes": req.Nodes,
	}
	if req.Desc != "" {
		body["desc"] = req.Desc
	}
	if req.HashOn != "" {
		body["hash_on"] = req.HashOn
	}
	if req.Key != "" {
		body["key"] = req.Key
	}
	if req.Scheme != "" {
		body["scheme"] = req.Scheme
	}
	if req.PassHost != "" {
		body["pass_host"] = req.PassHost
	}
	if req.UpstreamHost != "" {
		body["upstream_host"] = req.UpstreamHost
	}
	if req.Retries > 0 {
		body["retries"] = req.Retries
	}
	if req.RetryTimeout > 0 {
		body["retry_timeout"] = req.RetryTimeout
	}
	if req.Timeout != nil {
		body["timeout"] = req.Timeout
	}
	return body
}

// parseUpstreamList 解析 Apisix Upstream 列表响应
func parseUpstreamList(data []byte) ([]Upstream, error) {
	var raw struct {
		List []struct {
			Value Upstream `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Upstream 列表失败: %w", err)
	}
	result := make([]Upstream, 0, len(raw.List))
	for _, item := range raw.List {
		result = append(result, item.Value)
	}
	return result, nil
}

// parseSingleUpstream 解析单个 Upstream 响应
func parseSingleUpstream(data []byte) (*Upstream, error) {
	var raw struct {
		Value Upstream `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Upstream 详情失败: %w", err)
	}
	return &raw.Value, nil
}

// HasUpstreamNodes 判断 Upstream 是否配置了有效节点
func HasUpstreamNodes(nodes any) bool {
	switch v := nodes.(type) {
	case []any:
		return len(v) > 0
	case []map[string]any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	case map[string]int:
		return len(v) > 0
	case map[string]float64:
		return len(v) > 0
	default:
		return false
	}
}
