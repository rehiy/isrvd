// Package caddy 提供基于 Caddy Admin API 的 Go 客户端。
package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"
)

const (
	defaultBaseURL = "http://localhost:2019"
	defaultTimeout = 30 * time.Second
	ctJSON         = "application/json"
)

// Client Caddy Admin API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 Caddy Admin API 客户端
//
// baseURL 留空使用默认 http://localhost:2019。
func NewClient(baseURL string) *Client {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// ConfigLoad 使用 POST /load 替换全部配置（原子替换）
func (c *Client) ConfigLoad(ctx context.Context, cfg *Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化 caddy 配置失败: %w", err)
	}
	_, err = c.do(ctx, http.MethodPost, "/load", body, ctJSON)
	return err
}

// ConfigLoadRaw 使用任意 JSON 字节加载配置
func (c *Client) ConfigLoadRaw(ctx context.Context, raw []byte) error {
	_, err := c.do(ctx, http.MethodPost, "/load", raw, ctJSON)
	return err
}

// ConfigAll 获取当前完整配置
func (c *Client) ConfigAll(ctx context.Context) (*Config, error) {
	raw, err := c.ConfigRaw(ctx, "")
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return &Config{}, nil
	}
	var out Config
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析 caddy 配置失败: %w", err)
	}
	return &out, nil
}

// ConfigRaw 获取指定路径的配置原始 JSON，path 为空表示根配置
func (c *Client) ConfigRaw(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, joinConfig(path), nil, "")
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, contentType string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("创建 caddy 请求失败: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logman.Error("Caddy admin request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("请求 Caddy admin 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusMultipleChoices {
		logman.Error("Caddy admin error", "method", method, "path", path, "status", resp.StatusCode, "body", string(raw))
		return nil, fmt.Errorf("caddy %s %s 状态码 %d: %s", method, path, resp.StatusCode, string(raw))
	}
	return raw, nil
}

// joinConfig 把 path 拼接到 /config 之下，并对每段做 PathEscape
func joinConfig(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "/config/"
	}
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return "/config/" + strings.Join(segs, "/")
}
