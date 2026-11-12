package caddy

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"isrvd/pkgs/caddy"
)

var (
	ErrInvalidRoutePosition  = errors.New("路由位置超出范围")
	ErrRouteRevisionRequired = errors.New("调整路由位置前请先获取路由列表版本")
	ErrRouteRevisionChanged  = errors.New("路由列表已变化，请刷新后重试")
)

// RouteSnapshot 路由列表及版本；通过 snapshot=true 查询，不改变默认列表响应。
type RouteSnapshot struct {
	Routes   []RouteView `json:"routes"`
	Revision string      `json:"revision"`
}

// RouteMoveRequest 指定移动后的数组下标，0 表示第一条。
type RouteMoveRequest struct {
	Index    *int   `json:"index" binding:"required"`
	Revision string `json:"revision"`
}

// RouteMove 在一次配置重载中移动路由，保留路由的所有原生字段。
func (s *Service) RouteMove(ctx context.Context, server string, index int, req RouteMoveRequest) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if err := checkRouteRevision(srv.Routes, req.Revision); err != nil {
			return err
		}
		route, err := getRoute(srv, index)
		if err != nil {
			return err
		}
		if req.Index == nil || *req.Index < 0 || *req.Index >= len(srv.Routes) {
			return ErrInvalidRoutePosition
		}
		moved := *route
		srv.Routes = slices.Delete(srv.Routes, index, index+1)
		srv.Routes = slices.Insert(srv.Routes, *req.Index, moved)
		return nil
	})
}

// ─── 辅助函数 ───

// routeRevision 与列表来自同一快照，包含执行顺序及路由的全部原生字段。
func routeRevision(routes []caddy.Route) (string, error) {
	// 未配置 routes 和空数组都代表空列表。
	if routes == nil {
		routes = []caddy.Route{}
	}
	raw, err := json.Marshal(routes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}

func checkRouteRevision(routes []caddy.Route, revision string) error {
	if revision == "" {
		return ErrRouteRevisionRequired
	}
	current, err := routeRevision(routes)
	if err != nil {
		return err
	}
	if revision != current {
		return ErrRouteRevisionChanged
	}
	return nil
}
