package caddy

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"isrvd/pkgs/caddy"
)

var ErrRouteNotFound = errors.New("路由不存在")

// ─── 路由 CRUD ───

// RouteView 路由响应视图，在 caddy.Route 基础上附加列表下标
type RouteView struct {
	Index int `json:"index"` // 路由在列表中的下标（用于定位/删除）
	caddy.Route
}

// MarshalJSON 显式附加下标，避免内嵌 Route 的序列化方法吞掉视图字段。
func (v RouteView) MarshalJSON() ([]byte, error) {
	route, err := v.Route.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(route, &fields); err != nil {
		return nil, err
	}
	index, err := json.Marshal(v.Index)
	if err != nil {
		return nil, err
	}
	// 下标由服务端生成，覆盖 Extras 中可能存在的同名字段。
	fields["index"] = index
	return json.Marshal(fields)
}

// RouteList 列出指定 server 的所有路由
func (s *Service) RouteList(ctx context.Context, server string) ([]RouteView, string, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return nil, "", err
	}
	srv, err := s.serverConfig(ctx, server)
	if err != nil {
		return nil, "", err
	}
	out := make([]RouteView, len(srv.Routes))
	for i, r := range srv.Routes {
		out[i] = RouteView{Index: i, Route: r}
	}
	revision, err := routeRevision(srv.Routes)
	return out, revision, err
}

// RouteInspect 获取单条路由
func (s *Service) RouteInspect(ctx context.Context, server string, index int) (*RouteView, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return nil, err
	}
	srv, err := s.serverConfig(ctx, server)
	if err != nil {
		return nil, err
	}
	route, err := getRoute(srv, index)
	if err != nil {
		return nil, err
	}
	v := &RouteView{Index: index, Route: *route}
	return v, nil
}

// RouteCreate 创建路由；未指定位置时追加，指定位置时校验列表版本后插入。
func (s *Service) RouteCreate(ctx context.Context, server string, req caddy.Route, position *int, revision string) (int, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return -1, err
	}
	index := -1
	err = s.serverMutate(ctx, server, func(srv *caddy.HTTPServer) error {
		index = len(srv.Routes)
		if position != nil {
			if err := checkRouteRevision(srv.Routes, revision); err != nil {
				return err
			}
			index = *position
			if index < 0 || index > len(srv.Routes) {
				return ErrInvalidRoutePosition
			}
		}
		srv.Routes = slices.Insert(srv.Routes, index, req)
		return nil
	})
	if err != nil {
		return -1, err
	}
	return index, nil
}

// RouteUpdate 更新指定下标的路由
func (s *Service) RouteUpdate(ctx context.Context, server string, index int, req caddy.Route, revision string) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.routeMutate(ctx, server, index, revision, false, func(_ *caddy.HTTPServer, route *caddy.Route) error {
		*route = req
		return nil
	})
}

// RouteDelete 删除指定下标的路由
func (s *Service) RouteDelete(ctx context.Context, server string, index int, revision string) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.routeMutate(ctx, server, index, revision, false, func(srv *caddy.HTTPServer, _ *caddy.Route) error {
		srv.Routes = slices.Delete(srv.Routes, index, index+1)
		return nil
	})
}
