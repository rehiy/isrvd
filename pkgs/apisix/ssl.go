package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SSL Apisix SSL 证书信息
type SSL struct {
	ID         string   `json:"id,omitempty"`     // 证书 ID（创建时由 Apisix 自动生成）
	Snis       []string `json:"snis"`             // SNI 域名列表（用于 HTTPS 证书匹配）
	Cert       string   `json:"cert,omitempty"`   // PEM 格式证书
	Key        string   `json:"key,omitempty"`    // PEM 格式私钥
	Status     *int     `json:"status,omitempty"` // 状态：1=启用，0=禁用（默认为 1）
	CreateTime int64    `json:"create_time"`      // 创建时间（Unix 时间戳，只读）
	UpdateTime int64    `json:"update_time"`      // 更新时间（Unix 时间戳，只读）
}

// SSLList 获取所有 SSL 证书列表
func (c *Client) SSLList(ctx context.Context) ([]SSL, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/ssls", nil)
	if err != nil {
		return nil, err
	}
	return parseSSLList(data)
}

// SSLInspect 获取单个 SSL 证书详情
func (c *Client) SSLInspect(ctx context.Context, sslID string) (*SSL, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/ssls/"+url.PathEscape(sslID), nil)
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// SSLCreate 创建 SSL 证书
func (c *Client) SSLCreate(ctx context.Context, req SSL) (*SSL, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/ssls", buildSSLBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// SSLUpdate 更新 SSL 证书
func (c *Client) SSLUpdate(ctx context.Context, sslID string, req SSL) (*SSL, error) {
	data, err := c.doRequest(ctx, http.MethodPatch, "/ssls/"+url.PathEscape(sslID), buildSSLBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// SSLDelete 删除 SSL 证书
func (c *Client) SSLDelete(ctx context.Context, sslID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/ssls/"+url.PathEscape(sslID), nil)
	return err
}

// --- 辅助函数 ---

// buildSSLBody 将 SSL 转换为 Apisix API 请求体
func buildSSLBody(req SSL) map[string]any {
	body := make(map[string]any)
	if len(req.Snis) > 0 {
		body["snis"] = req.Snis
	}
	if req.Cert != "" {
		body["cert"] = req.Cert
	}
	if req.Key != "" {
		body["key"] = req.Key
	}
	if req.Status != nil {
		body["status"] = *req.Status
	}
	return body
}

// parseSSLList 解析 Apisix SSL 列表响应
func parseSSLList(data []byte) ([]SSL, error) {
	var raw struct {
		List []struct {
			Value SSL `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 SSL 证书列表失败: %w", err)
	}
	result := make([]SSL, 0, len(raw.List))
	for _, item := range raw.List {
		result = append(result, item.Value)
	}
	return result, nil
}

// parseSingleSSL 解析单个 SSL 证书响应
func parseSingleSSL(data []byte) (*SSL, error) {
	var raw struct {
		Value SSL `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 SSL 证书详情失败: %w", err)
	}
	return &raw.Value, nil
}
