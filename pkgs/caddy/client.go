// Package caddy 提供基于 Caddy Admin API 的 Go 客户端。
package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// HTTPError 表示 Caddy Admin API 返回的非成功 HTTP 响应。
//
// 调用方可通过 errors.As 获取上游状态码，并映射为合适的业务响应。
type HTTPError struct {
	Method     string // 触发错误的 HTTP 方法
	Path       string // Caddy Admin API 路径
	StatusCode int    // 上游返回的 HTTP 状态码
	Body       string // 上游响应体原文
}

// Error 实现 error 接口。
func (e *HTTPError) Error() string {
	return fmt.Sprintf("caddy %s %s 状态码 %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}

var ErrConfigChanged = errors.New("Caddy 配置已变化，请刷新后重试")

const rootConfigPath = "/config/"

// Client Caddy Admin API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	configMu   sync.RWMutex
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

// ConfigLoadRaw 使用任意 JSON 字节加载配置。
func (c *Client) ConfigLoadRaw(ctx context.Context, raw []byte) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	_, _, err := c.do(ctx, http.MethodPost, "/load", raw, "")
	return err
}

// ConfigMutate 在同一临界区内读取、修改并加载完整配置。
//
// fn 返回错误时不会向 Caddy 提交配置。
func (c *Client) ConfigMutate(ctx context.Context, fn func(*Config) error) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	raw, etag, err := c.do(ctx, http.MethodGet, rootConfigPath, nil, "")
	if err != nil {
		return err
	}
	if etag == "" {
		return errors.New("Caddy Admin API 未返回配置版本")
	}
	cfg, err := decodeConfig(raw)
	if err != nil {
		return err
	}
	if err := fn(cfg); err != nil {
		return err
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化 caddy 配置失败: %w", err)
	}
	_, _, err = c.do(ctx, http.MethodPost, rootConfigPath, body, etag)
	var status *HTTPError
	if errors.As(err, &status) && status.StatusCode == http.StatusPreconditionFailed {
		return ErrConfigChanged
	}
	return err
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

// ─── 内部方法 ───

func (c *Client) configAll(ctx context.Context) (*Config, error) {
	raw, err := c.configRaw(ctx, "")
	if err != nil {
		return nil, err
	}
	return decodeConfig(raw)
}

func decodeConfig(raw []byte) (*Config, error) {
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
	cfgPath := rootConfigPath
	if path != "" {
		segs := strings.Split(path, "/")
		for i, s := range segs {
			segs[i] = url.PathEscape(s)
		}
		cfgPath += strings.Join(segs, "/")
	}
	raw, _, err := c.do(ctx, http.MethodGet, cfgPath, nil, "")
	return raw, err
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, ifMatch string) ([]byte, string, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, "", fmt.Errorf("创建 caddy 请求失败: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", ctJSON)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
		// Caddy 重载会关闭旧连接。条件请求可由 HTTP 传输层重试，空值不发送给服务端。
		req.Header["Idempotency-Key"] = nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logman.Error("Caddy admin request failed", "method", method, "path", path, "error", err)
		return nil, "", fmt.Errorf("请求 Caddy admin 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取 Caddy admin 响应失败: %w", err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		logman.Error("Caddy admin error", "method", method, "path", path, "status", resp.StatusCode, "body", string(raw))
		return nil, "", &HTTPError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Body:       string(raw),
		}
	}
	return raw, resp.Header.Get("ETag"), nil
}
