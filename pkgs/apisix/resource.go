package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ─── 辅助函数 ───

type resourceValue[T any] struct {
	Value T `json:"value"`
}

// requestResourceList 读取资源列表，保留空列表的 [] JSON 语义。
func (c *Client) requestResourceList[T any](ctx context.Context, path, parseError string) ([]T, error) {
	var raw struct {
		List []resourceValue[T] `json:"list"`
	}
	if err := c.requestJSON(ctx, http.MethodGet, path, nil, &raw, parseError); err != nil {
		return nil, err
	}
	result := make([]T, len(raw.List))
	for i, item := range raw.List {
		result[i] = item.Value
	}
	return result, nil
}

// requestResource 发送资源请求并解析 value，资源字段转换与脱敏由调用方负责。
func (c *Client) requestResource[T any](ctx context.Context, method, path string, body any, parseError string) (*T, error) {
	var raw resourceValue[T]
	if err := c.requestJSON(ctx, method, path, body, &raw, parseError); err != nil {
		return nil, err
	}
	return &raw.Value, nil
}

func (c *Client) requestJSON(ctx context.Context, method, path string, body, result any, parseError string) error {
	data, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("%s: %w", parseError, err)
	}
	return nil
}
