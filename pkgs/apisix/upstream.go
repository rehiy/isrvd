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
	ID           string         `json:"id,omitempty"`            // Upstream ID（创建时由 Apisix 自动生成）
	Name         string         `json:"name"`                    // Upstream 名称
	Desc         string         `json:"desc,omitempty"`          // Upstream 描述
	Type         string         `json:"type"`                    // 负载均衡算法：roundrobin | chash | ewma | least_conn
	Nodes        any            `json:"nodes,omitempty"`         // 节点配置：map[string]int 格式，如 {"host:port": weight}
	HashOn       string         `json:"hash_on,omitempty"`       // chash 模式的哈希源：vars | header | cookie
	Key          string         `json:"key,omitempty"`           // chash 模式的哈希键（配合 hash_on 使用）
	Scheme       string         `json:"scheme,omitempty"`        // 转发协议：http | https | grpc | grpcs
	PassHost     string         `json:"pass_host,omitempty"`     // Host 传递方式：pass | node | rewrite
	UpstreamHost string         `json:"upstream_host,omitempty"` // pass_host=rewrite 时设置的上游 Host
	Retries      int            `json:"retries,omitempty"`       // 重试次数
	RetryTimeout int            `json:"retry_timeout,omitempty"` // 重试超时（秒）
	Timeout      map[string]any `json:"timeout,omitempty"`       // 超时配置：{"connect": 15, "send": 15, "read": 15}
	CreateTime   int64          `json:"create_time"`             // 创建时间（Unix 时间戳，只读）
	UpdateTime   int64          `json:"update_time"`             // 更新时间（Unix 时间戳，只读）
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
