package apisix

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSL Apisix SSL 证书信息
type SSL struct {
	ID         string   `json:"id,omitempty"`
	Snis       []string `json:"snis"`
	Cert       string   `json:"cert,omitempty"`
	Key        string   `json:"key,omitempty"`
	Status     *int     `json:"status,omitempty"`
	CreateTime int64    `json:"create_time"`
	UpdateTime int64    `json:"update_time"`
}

// ListSSLs 获取所有 SSL 证书列表
func (c *Client) ListSSLs() ([]SSL, error) {
	data, err := c.doRequest(http.MethodGet, "/ssls", nil)
	if err != nil {
		return nil, err
	}
	return parseSSLList(data)
}

// GetSSL 获取单个 SSL 证书详情
func (c *Client) GetSSL(sslID string) (*SSL, error) {
	data, err := c.doRequest(http.MethodGet, "/ssls/"+sslID, nil)
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// CreateSSL 创建 SSL 证书
func (c *Client) CreateSSL(req SSL) (*SSL, error) {
	data, err := c.doRequest(http.MethodPost, "/ssls", buildSSLBody(req, false))
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// UpdateSSL 更新 SSL 证书
func (c *Client) UpdateSSL(sslID string, req SSL) (*SSL, error) {
	data, err := c.doRequest(http.MethodPatch, "/ssls/"+sslID, buildSSLBody(req, true))
	if err != nil {
		return nil, err
	}
	return parseSingleSSL(data)
}

// DeleteSSL 删除 SSL 证书
func (c *Client) DeleteSSL(sslID string) error {
	_, err := c.doRequest(http.MethodDelete, "/ssls/"+sslID, nil)
	if err != nil {
		return err
	}
	return nil
}

// buildSSLBody 将 SSL 转换为 Apisix API 请求体
func buildSSLBody(req SSL, patch bool) map[string]any {
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
