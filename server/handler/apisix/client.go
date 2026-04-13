package apisix

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"isrvd/server/config"
)

// ========================
// Apisix 包内类型定义
// ========================

// ApisixConsumer Apisix Consumer 原始结构
type ApisixConsumer struct {
	Username   string         `json:"username"`
	Desc       string         `json:"desc"`
	Plugins    map[string]any `json:"plugins"`
	CreateTime int64          `json:"create_time"`
	UpdateTime int64          `json:"update_time"`
}

// ApisixRoute Apisix 路由原始结构
type ApisixRoute struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	URI             string         `json:"uri"`
	URIs            []string       `json:"uris"`
	Host            string         `json:"host"`
	Hosts           []string       `json:"hosts"`
	Desc            string         `json:"desc"`
	Status          int            `json:"status"`
	Priority        int            `json:"priority"`
	EnableWebsocket bool           `json:"enable_websocket"`
	PluginConfigID  string         `json:"plugin_config_id"`
	UpstreamID      string         `json:"upstream_id"`
	Upstream        map[string]any `json:"upstream"`
	Plugins         map[string]any `json:"plugins"`
	CreateTime      int64          `json:"create_time"`
	UpdateTime      int64          `json:"update_time"`
}

// ConsumerDTO Consumer 信息（对外暴露）
type ConsumerDTO struct {
	Username    string `json:"username"`
	APIKey      string `json:"api_key"`
	Description string `json:"description"`
	CreateTime  int64  `json:"create_time"`
	UpdateTime  int64  `json:"update_time"`
}

// RouteDTO 路由信息（对外暴露）
type RouteDTO struct {
	RouteID         string         `json:"route_id"`
	RouteName       string         `json:"route_name"`
	URI             string         `json:"uri,omitempty"`
	URIs            []string       `json:"uris,omitempty"`
	Host            string         `json:"host,omitempty"`
	Hosts           []string       `json:"hosts,omitempty"`
	Description     string         `json:"description,omitempty"`
	Status          int            `json:"status"`
	Priority        int            `json:"priority"`
	EnableWebsocket bool           `json:"enable_websocket"`
	PluginConfigID  string         `json:"plugin_config_id,omitempty"`
	UpstreamID      string         `json:"upstream_id,omitempty"`
	Upstream        map[string]any `json:"upstream,omitempty"`
	Plugins         map[string]any `json:"plugins,omitempty"`
	Consumers       []string       `json:"consumers,omitempty"`
	CreateTime      int64          `json:"create_time"`
	UpdateTime      int64          `json:"update_time"`
}

// PluginConfigDTO Plugin Config 信息（对外暴露）
type PluginConfigDTO struct {
	ID         string         `json:"id"`
	Desc       string         `json:"desc"`
	Plugins    map[string]any `json:"plugins,omitempty"`
	CreateTime int64          `json:"create_time"`
	UpdateTime int64          `json:"update_time"`
}

// UpstreamDTO Upstream 信息（对外暴露）
type UpstreamDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Type       string `json:"type"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

// RouteUpsertRequest 创建/更新路由的请求体
type RouteUpsertRequest struct {
	Name            string         `json:"name"`
	Desc            string         `json:"desc,omitempty"`
	URI             string         `json:"uri,omitempty"`
	URIs            []string       `json:"uris,omitempty"`
	Host            string         `json:"host,omitempty"`
	Hosts           []string       `json:"hosts,omitempty"`
	Status          int            `json:"status"`
	Priority        int            `json:"priority,omitempty"`
	EnableWebsocket bool           `json:"enable_websocket,omitempty"`
	PluginConfigID  string         `json:"plugin_config_id,omitempty"`
	UpstreamID      string         `json:"upstream_id,omitempty"`
	Upstream        map[string]any `json:"upstream,omitempty"`
	Plugins         map[string]any `json:"plugins,omitempty"`
}

// Client Apisix Admin API 客户端
type Client struct {
	baseURL    string
	adminKey   string
	httpClient *http.Client
}

// NewClient 创建 Apisix Admin API 客户端
func NewClient(cfg *config.ApisixConfig) *Client {
	return &Client{
		baseURL:  cfg.AdminURL,
		adminKey: cfg.AdminKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ========================
// 底层请求方法
// ========================

// doRequest 发送请求到 Apisix Admin API
func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("X-API-KEY", c.adminKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Apisix 失败: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Apisix 返回错误 [%d]: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// ========================
// Consumer 操作
// ========================

// ListConsumers 获取所有 Consumer 列表
func (c *Client) ListConsumers(ctx context.Context) ([]ConsumerDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/consumers", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Consumers 失败: %w", err)
	}

	var raw struct {
		List []struct {
			Value ApisixConsumer `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Consumer 列表失败: %w", err)
	}

	result := make([]ConsumerDTO, 0, len(raw.List))
	for _, item := range raw.List {
		v := item.Value
		apiKey := maskAPIKey(extractKeyAuthKey(v.Plugins))
		result = append(result, ConsumerDTO{
			Username:    v.Username,
			APIKey:      apiKey,
			Description: v.Desc,
			CreateTime:  v.CreateTime,
			UpdateTime:  v.UpdateTime,
		})
	}
	return result, nil
}

// GetConsumerRawKey 获取指定 Consumer 的完整（未脱敏）API Key
func (c *Client) GetConsumerRawKey(ctx context.Context, username string) (string, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/consumers/"+username, nil)
	if err != nil {
		return "", err
	}

	var raw struct {
		Value ApisixConsumer `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("解析 Consumer 失败: %w", err)
	}

	key := extractKeyAuthKey(raw.Value.Plugins)
	if key == "" {
		return "", fmt.Errorf("该用户尚未配置 API Key")
	}
	return key, nil
}

// CreateOrUpdateConsumer 创建或更新 Consumer（key-auth 插件）
func (c *Client) CreateOrUpdateConsumer(ctx context.Context, username, apiKey, description string) error {
	body := map[string]any{
		"username": username,
		"desc":     description,
		"plugins": map[string]any{
			"key-auth": map[string]any{
				"key": apiKey,
			},
		},
	}
	_, err := c.doRequest(ctx, http.MethodPut, "/consumers/"+username, body)
	if err != nil {
		return fmt.Errorf("创建/更新 Consumer 失败: %w", err)
	}
	return nil
}

// DeleteConsumer 删除指定 Consumer
func (c *Client) DeleteConsumer(ctx context.Context, username string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/consumers/"+username, nil)
	if err != nil {
		return fmt.Errorf("删除 Consumer 失败: %w", err)
	}
	return nil
}

// ========================
// 路由操作
// ========================

// routeToDTO 将 Apisix 原始路由结构转换为 RouteDTO
func routeToDTO(v ApisixRoute) RouteDTO {
	return RouteDTO{
		RouteID:         v.ID,
		RouteName:       v.Name,
		URI:             v.URI,
		URIs:            v.URIs,
		Host:            v.Host,
		Hosts:           v.Hosts,
		Description:     v.Desc,
		Status:          v.Status,
		Priority:        v.Priority,
		EnableWebsocket: v.EnableWebsocket,
		PluginConfigID:  v.PluginConfigID,
		UpstreamID:      v.UpstreamID,
		Upstream:        v.Upstream,
		Plugins:         v.Plugins,
		CreateTime:      v.CreateTime,
		UpdateTime:      v.UpdateTime,
	}
}

// parseRouteList 解析 Apisix 路由列表响应
func parseRouteList(data []byte) ([]ApisixRoute, error) {
	var raw struct {
		List []struct {
			Value ApisixRoute `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析路由列表失败: %w", err)
	}
	routes := make([]ApisixRoute, 0, len(raw.List))
	for _, item := range raw.List {
		routes = append(routes, item.Value)
	}
	return routes, nil
}

// parseSingleRoute 解析单个路由响应
func parseSingleRoute(data []byte) (*RouteDTO, error) {
	var raw struct {
		Value ApisixRoute `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析路由详情失败: %w", err)
	}
	dto := routeToDTO(raw.Value)
	return &dto, nil
}

// ListRoutes 获取所有路由列表
func (c *Client) ListRoutes(ctx context.Context) ([]RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("获取路由失败: %w", err)
	}
	routes, err := parseRouteList(data)
	if err != nil {
		return nil, err
	}
	result := make([]RouteDTO, 0, len(routes))
	for _, v := range routes {
		result = append(result, routeToDTO(v))
	}
	return result, nil
}

// GetRoute 获取单条路由详情
func (c *Client) GetRoute(ctx context.Context, routeID string) (*RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes/"+routeID, nil)
	if err != nil {
		return nil, fmt.Errorf("获取路由详情失败: %w", err)
	}
	return parseSingleRoute(data)
}

// CreateRoute 创建路由
func (c *Client) CreateRoute(ctx context.Context, req RouteUpsertRequest) (*RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/routes", buildRouteBody(req))
	if err != nil {
		return nil, fmt.Errorf("创建路由失败: %w", err)
	}
	return parseSingleRoute(data)
}

// UpdateRoute 更新路由
func (c *Client) UpdateRoute(ctx context.Context, routeID string, req RouteUpsertRequest) (*RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/routes/"+routeID, buildRouteBody(req))
	if err != nil {
		return nil, fmt.Errorf("更新路由失败: %w", err)
	}
	return parseSingleRoute(data)
}

// PatchRouteStatus 仅更新路由的启用/禁用状态（1=启用 0=禁用）
func (c *Client) PatchRouteStatus(ctx context.Context, routeID string, status int) error {
	body := map[string]any{"status": status}
	_, err := c.doRequest(ctx, http.MethodPatch, "/routes/"+routeID, body)
	if err != nil {
		return fmt.Errorf("更新路由状态失败: %w", err)
	}
	return nil
}

// DeleteRoute 删除路由
func (c *Client) DeleteRoute(ctx context.Context, routeID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/routes/"+routeID, nil)
	if err != nil {
		return fmt.Errorf("删除路由失败: %w", err)
	}
	return nil
}

// ========================
// 白名单操作
// ========================

// WhitelistRoutes 获取管控路由列表（仅返回同时配置了 key-auth 和 consumer-restriction 的路由）
func (c *Client) WhitelistRoutes(ctx context.Context) ([]RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("获取路由失败: %w", err)
	}
	routes, err := parseRouteList(data)
	if err != nil {
		return nil, err
	}
	result := make([]RouteDTO, 0, len(routes))
	for _, v := range routes {
		if !hasPlugin(v.Plugins, "key-auth") || !hasPlugin(v.Plugins, "consumer-restriction") {
			continue
		}
		result = append(result, routeToDTO(v))
	}
	return result, nil
}

// GetRouteWhitelist 获取所有路由的 consumer-restriction 白名单
func (c *Client) GetRouteWhitelist(ctx context.Context) ([]RouteDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("获取路由列表失败: %w", err)
	}
	routes, err := parseRouteList(data)
	if err != nil {
		return nil, err
	}

	result := make([]RouteDTO, 0)
	for _, v := range routes {
		consumers := extractConsumerRestrictionWhitelist(v.Plugins)
		if len(consumers) == 0 {
			continue
		}
		result = append(result, RouteDTO{
			RouteID:     v.ID,
			RouteName:   v.Name,
			URI:         v.URI,
			URIs:        v.URIs,
			Host:        v.Host,
			Hosts:       v.Hosts,
			Consumers:   consumers,
			Description: v.Desc,
		})
	}
	return result, nil
}

// getRouteConsumers 获取指定路由的白名单消费者列表
func (c *Client) getRouteConsumers(ctx context.Context, routeID string) ([]string, error) {
	whitelist, err := c.GetRouteWhitelist(ctx)
	if err != nil {
		return nil, err
	}
	for _, wl := range whitelist {
		if wl.RouteID == routeID {
			return wl.Consumers, nil
		}
	}
	return []string{}, nil
}

// RemoveConsumerFromRouteWhitelist 从路由的白名单中移除 consumer
func (c *Client) RemoveConsumerFromRouteWhitelist(ctx context.Context, routeID, consumerName string) error {
	consumers, err := c.getRouteConsumers(ctx, routeID)
	if err != nil {
		return err
	}
	newConsumers := make([]string, 0, len(consumers))
	found := false
	for _, name := range consumers {
		if name == consumerName {
			found = true
			continue
		}
		newConsumers = append(newConsumers, name)
	}
	if !found {
		return fmt.Errorf("用户 %s 不在路由 %s 的白名单中", consumerName, routeID)
	}
	return c.UpdateRouteConsumerRestriction(ctx, routeID, newConsumers)
}

// UpdateRouteConsumerRestriction 更新路由的 consumer-restriction 白名单
func (c *Client) UpdateRouteConsumerRestriction(ctx context.Context, routeID string, consumers []string) error {
	routeData, err := c.doRequest(ctx, http.MethodGet, "/routes/"+routeID, nil)
	if err != nil {
		return fmt.Errorf("获取路由详情失败: %w", err)
	}
	var raw struct {
		Value map[string]any `json:"value"`
	}
	if err := json.Unmarshal(routeData, &raw); err != nil {
		return fmt.Errorf("解析路由详情失败: %w", err)
	}

	route := raw.Value
	plugins, _ := route["plugins"].(map[string]any)
	if plugins == nil {
		plugins = make(map[string]any)
	}

	if len(consumers) > 0 {
		plugins["consumer-restriction"] = map[string]any{"whitelist": consumers}
	} else {
		delete(plugins, "consumer-restriction")
	}

	route["plugins"] = plugins
	delete(route, "create_time")
	delete(route, "update_time")

	_, err = c.doRequest(ctx, http.MethodPut, "/routes/"+routeID, route)
	if err != nil {
		return fmt.Errorf("更新路由白名单失败: %w", err)
	}
	return nil
}

// ========================
// 其他资源操作
// ========================

// ListPluginConfigs 获取所有 Plugin Config 列表
func (c *Client) ListPluginConfigs(ctx context.Context) ([]PluginConfigDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/plugin_configs", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Plugin Config 列表失败: %w", err)
	}
	var raw struct {
		List []struct {
			Value struct {
				ID         string         `json:"id"`
				Desc       string         `json:"desc"`
				Plugins    map[string]any `json:"plugins"`
				CreateTime int64          `json:"create_time"`
				UpdateTime int64          `json:"update_time"`
			} `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Plugin Config 列表失败: %w", err)
	}
	result := make([]PluginConfigDTO, 0, len(raw.List))
	for _, item := range raw.List {
		v := item.Value
		result = append(result, PluginConfigDTO{
			ID:         v.ID,
			Desc:       v.Desc,
			Plugins:    v.Plugins,
			CreateTime: v.CreateTime,
			UpdateTime: v.UpdateTime,
		})
	}
	return result, nil
}

// ListPlugins 获取 Apisix 中所有可用插件及其 schema
func (c *Client) ListPlugins(ctx context.Context) (map[string]any, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/plugins?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析插件列表失败: %w", err)
	}
	return result, nil
}

// ListUpstreams 获取所有 Upstream 列表
func (c *Client) ListUpstreams(ctx context.Context) ([]UpstreamDTO, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/upstreams", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Upstream 列表失败: %w", err)
	}
	var raw struct {
		List []struct {
			Value struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Desc       string `json:"desc"`
				Type       string `json:"type"`
				CreateTime int64  `json:"create_time"`
				UpdateTime int64  `json:"update_time"`
			} `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Upstream 列表失败: %w", err)
	}
	result := make([]UpstreamDTO, 0, len(raw.List))
	for _, item := range raw.List {
		v := item.Value
		result = append(result, UpstreamDTO{
			ID:         v.ID,
			Name:       v.Name,
			Desc:       v.Desc,
			Type:       v.Type,
			CreateTime: v.CreateTime,
			UpdateTime: v.UpdateTime,
		})
	}
	return result, nil
}

// ========================
// 工具方法
// ========================

// maskAPIKey 对 API Key 做脱敏处理
func maskAPIKey(key string) string {
	if len(key) == 0 {
		return ""
	}
	prefix := key
	if len(key) > 6 {
		prefix = key[:6]
	}
	suffix := ""
	if len(key) > 10 {
		suffix = key[len(key)-4:]
	}
	return prefix + "****" + suffix
}

// extractKeyAuthKey 从 plugins 中提取 key-auth 的 key
func extractKeyAuthKey(plugins map[string]any) string {
	if plugins == nil {
		return ""
	}
	keyAuth, ok := plugins["key-auth"].(map[string]any)
	if !ok {
		return ""
	}
	key, ok := keyAuth["key"].(string)
	if !ok {
		return ""
	}
	return key
}

// buildRouteBody 将 RouteUpsertRequest 转换为 Apisix API 请求体
func buildRouteBody(req RouteUpsertRequest) map[string]any {
	body := map[string]any{
		"name":     req.Name,
		"status":   req.Status,
		"priority": req.Priority,
	}
	if req.Desc != "" {
		body["desc"] = req.Desc
	}
	if req.EnableWebsocket {
		body["enable_websocket"] = true
	}
	if len(req.URIs) > 0 {
		body["uris"] = req.URIs
	} else if req.URI != "" {
		body["uri"] = req.URI
	}
	if len(req.Hosts) > 0 {
		body["hosts"] = req.Hosts
	} else if req.Host != "" {
		body["host"] = req.Host
	}
	if req.PluginConfigID != "" {
		body["plugin_config_id"] = req.PluginConfigID
	}
	if req.UpstreamID != "" {
		body["upstream_id"] = req.UpstreamID
	} else if req.Upstream != nil {
		body["upstream"] = req.Upstream
	}
	if req.Plugins != nil {
		body["plugins"] = req.Plugins
	}
	return body
}

// generateAPIKey 生成 32 字符的随机 API Key
func generateAPIKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// hasPlugin 检查插件列表中是否存在指定插件
func hasPlugin(plugins map[string]any, name string) bool {
	if plugins == nil {
		return false
	}
	_, ok := plugins[name]
	return ok
}

// extractConsumerRestrictionWhitelist 从 plugins 中提取 consumer-restriction 白名单
func extractConsumerRestrictionWhitelist(plugins map[string]any) []string {
	if plugins == nil {
		return []string{}
	}
	cr, ok := plugins["consumer-restriction"].(map[string]any)
	if !ok {
		return []string{}
	}
	wl, ok := cr["whitelist"].([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(wl))
	for _, v := range wl {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
