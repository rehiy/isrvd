package apisix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// RouteTimeout Apisix 路由超时配置
type RouteTimeout struct {
	Connect int `json:"connect,omitempty"`
	Send    int `json:"send,omitempty"`
	Read    int `json:"read,omitempty"`
}

// Route Apisix Route 信息
type Route struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	URI             string         `json:"uri,omitempty"`
	URIs            []string       `json:"uris,omitempty"`
	Host            string         `json:"host,omitempty"`
	Hosts           []string       `json:"hosts,omitempty"`
	Desc            string         `json:"desc,omitempty"`
	Status          int            `json:"status"`
	Priority        int            `json:"priority"`
	EnableWebsocket bool           `json:"enable_websocket"`
	PluginConfigID  string         `json:"plugin_config_id,omitempty"`
	UpstreamID      string         `json:"upstream_id,omitempty"`
	Upstream        map[string]any `json:"upstream,omitempty"`
	Plugins         map[string]any `json:"plugins,omitempty"`
	Consumers       []string       `json:"consumers,omitempty"`
	Timeout         *RouteTimeout  `json:"timeout,omitempty"`
	CreateTime      int64          `json:"create_time"`
	UpdateTime      int64          `json:"update_time"`
}

// RouteList 获取所有路由列表（不过滤插件，用于路由管理页面展示）
func (c *Client) RouteList(ctx context.Context) ([]Route, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes", nil)
	if err != nil {
		return nil, err
	}
	return parseRouteList(data)
}

// RouteInspect 获取单条路由详情
func (c *Client) RouteInspect(ctx context.Context, routeID string) (*Route, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes/"+url.PathEscape(routeID), nil)
	if err != nil {
		return nil, err
	}
	return parseSingleRoute(data)
}

// RouteCreate 创建路由
func (c *Client) RouteCreate(ctx context.Context, req Route) (*Route, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/routes", buildRouteBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleRoute(data)
}

// RouteUpdate 更新路由
func (c *Client) RouteUpdate(ctx context.Context, routeID string, req Route) (*Route, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/routes/"+url.PathEscape(routeID), buildRouteBody(req))
	if err != nil {
		return nil, err
	}
	return parseSingleRoute(data)
}

// RouteStatusPatch 仅更新路由的启用/禁用状态（1=启用 0=禁用）
func (c *Client) RouteStatusPatch(ctx context.Context, routeID string, status int) error {
	body := map[string]any{"status": status}
	_, err := c.doRequest(ctx, http.MethodPatch, "/routes/"+url.PathEscape(routeID), body)
	return err
}

// RouteDelete 删除路由
func (c *Client) RouteDelete(ctx context.Context, routeID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/routes/"+url.PathEscape(routeID), nil)
	return err
}

// RouteWhitelistInspect 获取所有路由的 consumer-restriction 白名单
func (c *Client) RouteWhitelistInspect(ctx context.Context) ([]Route, error) {
	routes, err := c.fetchRoutes(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Route, 0)
	for _, v := range routes {
		consumers := pluginConsumerRestrictionWhitelist(v.Plugins)
		if len(consumers) == 0 {
			continue
		}
		v.Consumers = consumers
		result = append(result, v)
	}
	return result, nil
}

// getRouteConsumers 获取指定路由的白名单消费者列表
func (c *Client) getRouteConsumers(ctx context.Context, routeID string) ([]string, error) {
	whitelist, err := c.RouteWhitelistInspect(ctx)
	if err != nil {
		return nil, err
	}
	for _, wl := range whitelist {
		if wl.ID == routeID {
			return wl.Consumers, nil
		}
	}
	return []string{}, nil
}

// RouteWhitelistUserDelete 从路由的白名单中移除 consumer
func (c *Client) RouteWhitelistUserDelete(ctx context.Context, routeID, consumerName string) error {
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
	return c.RouteConsumerRestrictionUpdate(ctx, routeID, newConsumers, nil)
}

// RouteConsumerRestrictionUpdate 更新路由的 consumer-restriction 白名单
func (c *Client) RouteConsumerRestrictionUpdate(ctx context.Context, routeID string, consumers []string, keyAuth map[string]any) error {
	routeData, err := c.doRequest(ctx, http.MethodGet, "/routes/"+url.PathEscape(routeID), nil)
	if err != nil {
		return err
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
		if keyAuth != nil {
			plugins["key-auth"] = keyAuth
		}
		plugins["consumer-restriction"] = map[string]any{"whitelist": consumers}
	} else {
		delete(plugins, "consumer-restriction")
	}
	route["plugins"] = plugins
	delete(route, "create_time")
	delete(route, "update_time")
	_, err = c.doRequest(ctx, http.MethodPut, "/routes/"+url.PathEscape(routeID), route)
	return err
}

// fetchRoutes 拉取全量路由（内部复用，避免重复 HTTP 调用）
func (c *Client) fetchRoutes(ctx context.Context) ([]Route, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/routes", nil)
	if err != nil {
		return nil, err
	}
	return parseRouteList(data)
}

// --- 辅助函数 ---

// buildRouteBody 将 Route 转换为 Apisix API 请求体
func buildRouteBody(req Route) map[string]any {
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
	// URI / URIs
	if len(req.URIs) > 0 {
		body["uris"] = req.URIs
	} else if req.URI != "" {
		body["uri"] = req.URI
	}
	// Host / Hosts
	if len(req.Hosts) > 0 {
		body["hosts"] = req.Hosts
	} else if req.Host != "" {
		body["host"] = req.Host
	}
	// Plugin Config
	if req.PluginConfigID != "" {
		body["plugin_config_id"] = req.PluginConfigID
	}
	// Upstream
	if req.UpstreamID != "" {
		body["upstream_id"] = req.UpstreamID
	} else if req.Upstream != nil {
		body["upstream"] = req.Upstream
	}
	// Plugins
	if req.Plugins != nil {
		body["plugins"] = req.Plugins
	}
	// Timeout
	if req.Timeout != nil {
		body["timeout"] = req.Timeout
	}
	return body
}

// parseRouteList 解析 Apisix 路由列表响应
func parseRouteList(data []byte) ([]Route, error) {
	var raw struct {
		List []struct {
			Value Route `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析路由列表失败: %w", err)
	}
	routes := make([]Route, 0, len(raw.List))
	for _, item := range raw.List {
		routes = append(routes, item.Value)
	}
	return routes, nil
}

// parseSingleRoute 解析单个路由响应
func parseSingleRoute(data []byte) (*Route, error) {
	var raw struct {
		Value Route `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析路由详情失败: %w", err)
	}
	return &raw.Value, nil
}
