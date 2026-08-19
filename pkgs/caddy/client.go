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
	"sync"
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
	configMu   sync.RWMutex
}

// HTTPError 表示 Caddy Admin API 返回的非成功 HTTP 响应。
//
// 调用方可通过 errors.As 获取上游状态码，并映射为合适的业务响应。
type HTTPError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

// Error 实现 error 接口。
func (e *HTTPError) Error() string {
	return fmt.Sprintf("caddy %s %s 状态码 %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
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
	c.configMu.Lock()
	defer c.configMu.Unlock()
	return c.configLoad(ctx, cfg)
}

// ConfigLoadRaw 使用任意 JSON 字节加载配置。
func (c *Client) ConfigLoadRaw(ctx context.Context, raw []byte) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	return c.configLoadRaw(ctx, raw)
}

// ConfigMutate 在同一临界区内读取、修改并加载完整配置。
//
// fn 返回错误时不会向 Caddy 提交配置。
func (c *Client) ConfigMutate(ctx context.Context, fn func(*Config) error) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	cfg, err := c.configAll(ctx)
	if err != nil {
		return err
	}
	if err := fn(cfg); err != nil {
		return err
	}
	return c.configLoad(ctx, cfg)
}

// ConfigAll 获取当前完整配置。
func (c *Client) ConfigAll(ctx context.Context) (*Config, error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.configAll(ctx)
}

// ConfigRaw 获取指定路径的配置原始 JSON，path 为空表示根配置。
func (c *Client) ConfigRaw(ctx context.Context, path string) ([]byte, error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.configRaw(ctx, path)
}

// ─── 辅助函数 ───

func (c *Client) configLoad(ctx context.Context, cfg *Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化 caddy 配置失败: %w", err)
	}
	_, err = c.do(ctx, http.MethodPost, "/load", body, ctJSON)
	return err
}

func (c *Client) configLoadRaw(ctx context.Context, raw []byte) error {
	_, err := c.do(ctx, http.MethodPost, "/load", raw, ctJSON)
	return err
}

func (c *Client) configAll(ctx context.Context) (*Config, error) {
	raw, err := c.configRaw(ctx, "")
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

func (c *Client) configRaw(ctx context.Context, path string) ([]byte, error) {
	path = strings.Trim(path, "/")
	cfgPath := "/config/"
	if path != "" {
		segs := strings.Split(path, "/")
		for i, s := range segs {
			segs[i] = url.PathEscape(s)
		}
		cfgPath = "/config/" + strings.Join(segs, "/")
	}
	return c.do(ctx, http.MethodGet, cfgPath, nil, "")
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

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Caddy admin 响应失败: %w", err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		logman.Error("Caddy admin error", "method", method, "path", path, "status", resp.StatusCode, "body", string(raw))
		return nil, &HTTPError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Body:       string(raw),
		}
	}
	return raw, nil
}
