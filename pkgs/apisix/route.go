package apisix

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
)

// RouteTimeout Apisix 路由超时配置（单位：秒）
type RouteTimeout struct {
	Connect int `json:"connect,omitempty"` // 连接超时
	Send    int `json:"send,omitempty"`    // 发送超时
	Read    int `json:"read,omitempty"`    // 读取超时
}

// Route Apisix 路由信息
type Route struct {
	ID              string         `json:"id,omitempty"`               // 路由 ID（创建时由 Apisix 自动生成）
	Name            string         `json:"name"`                       // 路由名称
	URI             string         `json:"uri,omitempty"`              // 单个路由匹配 URI（与 URIs 二选一）
	URIs            []string       `json:"uris,omitempty"`             // 路由匹配 URI 列表
	Host            string         `json:"host,omitempty"`             // 单个匹配 Host（与 Hosts 二选一）
	Hosts           []string       `json:"hosts,omitempty"`            // 匹配 Host 列表
	Desc            string         `json:"desc,omitempty"`             // 路由描述
	Status          int            `json:"status"`                     // 路由状态：1=启用，0=禁用
	Priority        int            `json:"priority"`                   // 路由优先级（数值越大优先级越高）
	EnableWebsocket bool           `json:"enable_websocket"`           // 是否启用 WebSocket 代理
	PluginConfigID  string         `json:"plugin_config_id,omitempty"` // 引用的 PluginConfig ID
	ServiceID       string         `json:"service_id,omitempty"`       // 引用的 Service ID
	UpstreamID      string         `json:"upstream_id,omitempty"`      // 引用的 Upstream ID
	Upstream        map[string]any `json:"upstream,omitempty"`         // 内联 Upstream 配置（与 UpstreamID 二选一）
	Plugins         map[string]any `json:"plugins,omitempty"`          // 插件配置
	Consumers       []string       `json:"consumers,omitempty"`        // 白名单 Consumer 列表（只读，仅访问授权列表接口填充，请求中被忽略）
	Timeout         *RouteTimeout  `json:"timeout,omitempty"`          // 超时配置
	CreateTime      int64          `json:"create_time"`                // 创建时间（Unix 时间戳，只读）
	UpdateTime      int64          `json:"update_time"`                // 更新时间（Unix 时间戳，只读）
}

// RouteList 获取所有路由列表
func (c *Client) RouteList(ctx context.Context) ([]Route, error) {
	return c.requestResourceList[Route](ctx, "/routes", "解析路由列表失败")
}

// RouteInspect 获取单条路由详情
func (c *Client) RouteInspect(ctx context.Context, routeID string) (*Route, error) {
	return c.requestResource[Route](ctx, http.MethodGet, "/routes/"+url.PathEscape(routeID), nil, "解析路由详情失败")
}

// RouteCreate 创建路由
func (c *Client) RouteCreate(ctx context.Context, req Route) (*Route, error) {
	return c.requestResource[Route](ctx, http.MethodPost, "/routes", buildRouteBody(req), "解析路由详情失败")
}

// routeManagedFields 是 buildRouteBody 负责写入的字段；更新时先清除这些字段再覆盖，
// 其余字段（methods、remote_addrs、vars、labels 等）原样保留，避免编辑路由时静默丢失访问限制
var routeManagedFields = []string{
	"name", "desc", "status", "priority", "enable_websocket",
	"uri", "uris", "host", "hosts",
	"plugin_config_id", "upstream_id", "upstream", "plugins", "timeout",
	"create_time", "update_time",
}

// RouteUpdate 更新路由，保留请求模型之外的已有字段
func (c *Client) RouteUpdate(ctx context.Context, routeID string, req Route) (*Route, error) {
	path := "/routes/" + url.PathEscape(routeID)
	existing, err := c.requestResource[map[string]any](ctx, http.MethodGet, path, nil, "解析路由详情失败")
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	maps.Copy(body, *existing)
	maps.DeleteFunc(body, func(k string, _ any) bool {
		return slices.Contains(routeManagedFields, k)
	})
	maps.Copy(body, buildRouteBody(req))
	return c.requestResource[Route](ctx, http.MethodPut, path, body, "解析路由详情失败")
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

// RouteWhitelist 返回路由 consumer-restriction 插件中的当前白名单
func RouteWhitelist(route *Route) []string {
	if route == nil {
		return []string{}
	}
	return pluginConsumerRestrictionWhitelist(route.Plugins)
}

// RouteWhitelistInspect 获取所有路由的 consumer-restriction 白名单
func (c *Client) RouteWhitelistInspect(ctx context.Context) ([]Route, error) {
	routes, err := c.RouteList(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Route, 0, len(routes))
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

// ErrACLInherited 路由的 consumer-restriction 来自引用的 PluginConfig 或 Service，不在路由上修改
var ErrACLInherited = errors.New("授权由插件配置或服务管理，请在其中修改")

// RouteAuthInfo 路由的认证与授权配置来源（路由自身 plugins 与引用的 PluginConfig、Service 合并，路由同名插件优先）
type RouteAuthInfo struct {
	// ManagedElsewhere 路由自身没有 key-auth，而合并后已有认证类插件（APISIX 插件元数据 type=auth）。
	// 此时不应再写入路由上的 key-auth：叠加会变成必须同时通过，同名则会覆盖调用方正在使用的请求头；
	// 所需的认证方式也无法确定，不校验 Consumer 是否配置 key-auth。
	ManagedElsewhere bool
	// ACLInherited consumer-restriction 来自 PluginConfig 或 Service，路由自身没有
	ACLInherited bool
}

// RouteAuthInspect 分析路由的认证与授权配置来源
func (c *Client) RouteAuthInspect(ctx context.Context, route *Route) (*RouteAuthInfo, error) {
	shared := map[string]any{}
	if route.PluginConfigID != "" {
		pc, err := c.PluginConfigInspect(ctx, route.PluginConfigID)
		if err != nil {
			return nil, err
		}
		maps.Copy(shared, pc.Plugins)
	}
	if route.ServiceID != "" {
		svc, err := c.requestResource[struct {
			Plugins map[string]any `json:"plugins"`
		}](ctx, http.MethodGet, "/services/"+url.PathEscape(route.ServiceID), nil, "解析 Service 详情失败")
		if err != nil {
			return nil, err
		}
		maps.Copy(shared, svc.Plugins)
	}
	_, ownACL := route.Plugins["consumer-restriction"]
	_, sharedACL := shared["consumer-restriction"]
	info := &RouteAuthInfo{ACLInherited: !ownACL && sharedACL}
	if _, ownKey := route.Plugins["key-auth"]; ownKey {
		return info, nil
	}
	catalog, err := c.PluginList(ctx)
	if err != nil {
		return nil, err
	}
	for _, plugins := range []map[string]any{route.Plugins, shared} {
		for name := range plugins {
			if meta, _ := catalog[name].(map[string]any); meta["type"] == "auth" {
				info.ManagedElsewhere = true
			}
		}
	}
	return info, nil
}

// RouteConsumerRestrictionUpdate 更新路由的 consumer-restriction 白名单；consumers 为空表示删除授权。
// route 与 info 来自 RouteInspect 和 RouteAuthInspect，调用方需先拒绝 info.ACLInherited 的路由。
// 使用 PATCH 局部更新，只改动 key-auth 与 consumer-restriction，避免整条路由 GET+PUT 的覆盖问题。
func (c *Client) RouteConsumerRestrictionUpdate(ctx context.Context, route *Route, info *RouteAuthInfo, consumers []string, keyAuth map[string]any) error {
	base := "/routes/" + url.PathEscape(route.ID)
	if len(consumers) == 0 {
		// 合并 PATCH 中值为 null 表示删除该插件；对不存在的插件置 null 会被 APISIX 判为非法配置（400），
		// 因此只处理路由上实际存在的插件（认证由别处管理时路由上没有 key-auth，不会误删）
		del := map[string]any{}
		for _, name := range []string{"consumer-restriction", "key-auth"} {
			if _, ok := route.Plugins[name]; ok {
				del[name] = nil
			}
		}
		if len(del) == 0 {
			return nil
		}
		_, err := c.doRequest(ctx, http.MethodPatch, base, map[string]any{"plugins": del})
		return err
	}

	// 子路径 PATCH 整体替换该插件配置（而非深度合并）；consumer-restriction 写成纯白名单
	// （原有的黑名单、自定义拒绝响应等配置会被覆盖）
	if keyAuth != nil && !info.ManagedElsewhere {
		if _, err := c.doRequest(ctx, http.MethodPatch, base+"/plugins/key-auth", keyAuth); err != nil {
			return err
		}
	}
	_, err := c.doRequest(ctx, http.MethodPatch, base+"/plugins/consumer-restriction", map[string]any{"whitelist": consumers})
	return err
}

// ─── 辅助函数 ───

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
