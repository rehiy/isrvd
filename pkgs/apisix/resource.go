package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ─── 辅助函数 ───

// requestResourceList 读取资源列表，保留空列表的 [] JSON 语义。
func requestResourceList[T any](ctx context.Context, c *Client, path, parseError string) ([]T, error) {
	data, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var raw struct {
		List []struct {
			Value T `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", parseError, err)
	}
	result := make([]T, 0, len(raw.List))
	for _, item := range raw.List {
		result = append(result, item.Value)
	}
	return result, nil
}

// requestResource 发送资源请求并解析 value，资源字段转换与脱敏由调用方负责。
func requestResource[T any](ctx context.Context, c *Client, method, path string, body any, parseError string) (*T, error) {
	data, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Value T `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", parseError, err)
	}
	return &raw.Value, nil
}
