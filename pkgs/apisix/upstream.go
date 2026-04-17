package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Upstream Apisix Upstream 信息
type Upstream struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Type       string `json:"type"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

// ListUpstreams 获取所有 Upstream 列表
func (c *Client) ListUpstreams() ([]Upstream, error) {
	data, err := c.doRequest(http.MethodGet, "/upstreams", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Upstream 列表失败: %w", err)
	}
	return parseUpstreamList(data)
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
