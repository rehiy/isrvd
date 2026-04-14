package apisix

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rehiy/pango/logman"
	"github.com/rehiy/pango/request"
)

// Client APISIX Admin API 客户端
type Client struct {
	baseURL  string
	adminKey string
}

// NewClient 创建 APISIX Admin API 客户端
func NewClient(baseURL, adminKey string) *Client {
	return &Client{
		baseURL:  baseURL,
		adminKey: adminKey,
	}
}

// doRequest 发送请求到 APISIX Admin API
func (c *Client) doRequest(method, path string, body any) ([]byte, error) {
	var dataStr string
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		dataStr = string(data)
	}

	url := c.baseURL + path
	client := &request.Client{
		Method: method,
		Url:    url,
		Data:   dataStr,
		Headers: map[string]string{
			"X-API-KEY":    c.adminKey,
			"Content-Type": "application/json",
		},
		Timeout: 30 * time.Second,
	}

	respData, err := client.Request()
	if err != nil {
		logman.Error("APISIX request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("请求 APISIX 失败: %w", err)
	}

	return respData, nil
}
