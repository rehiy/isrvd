package apisix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rehiy/pango/logman"
)

// Client Apisix Admin API 客户端
type Client struct {
	baseURL    string
	adminKey   string
	httpClient *http.Client
}

// NewClient 创建 Apisix Admin API 客户端
func NewClient(baseURL, adminKey string) *Client {
	return &Client{
		baseURL:  baseURL,
		adminKey: adminKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest 发送请求到 Apisix Admin API
func (c *Client) doRequest(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("X-API-KEY", c.adminKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logman.Error("Apisix request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("请求 Apisix 失败: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logman.Error("Apisix request error", "method", method, "path", path, "status", resp.StatusCode, "body", string(respData))
		return nil, fmt.Errorf("Apisix 返回错误状态码 %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}
