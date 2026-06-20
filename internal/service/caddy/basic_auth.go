package caddy

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	pkgCaddy "isrvd/pkgs/caddy"
)

// BasicAuthUser basic_auth 账号视图（密码不回显）
type BasicAuthUser struct {
	Username string `json:"username"`
}

// BasicAuthRouteView 带账号列表的路由视图
type BasicAuthRouteView struct {
	Index         int                `json:"index"`         // 路由下标（主键）
	Name          string             `json:"name"`          // 路由名（仅展示）
	Realm         string             `json:"realm"`         // HTTP Basic realm
	ForwardHeader string             `json:"forwardHeader"` // 传递用户名的请求头；空表示未开启
	Users         []BasicAuthUser    `json:"users"`         // 账号列表（不含密码 hash）
	Handlers      []pkgCaddy.Handler `json:"handlers"`      // 其余 handler 链（只读展示）
}

// BasicAuthList 列出所有含 basic_auth 的路由
func (s *Service) BasicAuthList(ctx context.Context, server string) ([]BasicAuthRouteView, error) {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return []BasicAuthRouteView{}, nil
	}
	var out []BasicAuthRouteView
	for i, route := range srv.Routes {
		for _, h := range route.Handle {
			realm, accounts, ok := pkgCaddy.BasicAuthFromHandler(h)
			if !ok {
				continue
			}
			users := make([]BasicAuthUser, len(accounts))
			for j, a := range accounts {
				users[j] = BasicAuthUser{Username: a.Username}
			}
			var others []pkgCaddy.Handler
			for _, hh := range route.Handle {
				if hh["handler"] != "authentication" {
					others = append(others, hh)
				}
			}
			out = append(out, BasicAuthRouteView{
				Index:         i,
				Name:          route.ID,
				Realm:         realm,
				ForwardHeader: extractForwardHeader(route.Handle),
				Users:         users,
				Handlers:      others,
			})
			break
		}
	}
	return out, nil
}

// BasicAuthUserCreate 向指定路由添加一个 basic_auth 账号
//
// forwardHeader 非空时，在 handle 链中维护一个 headers handler，将
// {http.auth.user.id} 注入到指定请求头中传给后端；空字符串表示移除该注入。
func (s *Service) BasicAuthUserCreate(ctx context.Context, server string, routeIndex int, username, password, realm, forwardHeader string) error {
	if username == "" || password == "" {
		return fmt.Errorf("用户名和密码不能为空")
	}
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || routeIndex < 0 || routeIndex >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	route := &srv.Routes[routeIndex]

	authIdx := findAuthIndex(route)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if authIdx >= 0 {
		existingRealm, accounts, _ := pkgCaddy.BasicAuthFromHandler(route.Handle[authIdx])
		for _, a := range accounts {
			if a.Username == username {
				return fmt.Errorf("用户 %q 已存在", username)
			}
		}
		if realm == "" {
			realm = existingRealm
		}
		accounts = append(accounts, pkgCaddy.BasicAuthAccount{
			Username: username,
			Password: string(hash),
		})
		route.Handle[authIdx] = pkgCaddy.HandlerBasicAuth(realm, accounts)
	} else {
		newHandler := pkgCaddy.HandlerBasicAuth(realm, []pkgCaddy.BasicAuthAccount{
			{Username: username, Password: string(hash)},
		})
		route.Handle = append([]pkgCaddy.Handler{newHandler}, route.Handle...)
	}

	applyForwardHeader(route, forwardHeader)

	return s.client.ConfigLoad(ctx, cfg)
}

// BasicAuthUserDelete 从指定路由移除一个账号
//
// 若移除后账号列表为空，则同时删除 authentication handler 和 forward-user headers handler。
func (s *Service) BasicAuthUserDelete(ctx context.Context, server string, routeIndex int, username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || routeIndex < 0 || routeIndex >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	route := &srv.Routes[routeIndex]

	authIdx := findAuthIndex(route)
	if authIdx < 0 {
		return fmt.Errorf("该路由未配置 Basic Auth")
	}

	realm, accounts, _ := pkgCaddy.BasicAuthFromHandler(route.Handle[authIdx])
	filtered := accounts[:0]
	for _, a := range accounts {
		if a.Username != username {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == len(accounts) {
		return fmt.Errorf("用户 %q 不存在", username)
	}

	if len(filtered) == 0 {
		route.Handle = append(route.Handle[:authIdx], route.Handle[authIdx+1:]...)
		applyForwardHeader(route, "")
	} else {
		route.Handle[authIdx] = pkgCaddy.HandlerBasicAuth(realm, filtered)
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// BasicAuthConfigUpdate 更新已有认证路由的 realm 和 forwardHeader，不修改账号列表
func (s *Service) BasicAuthConfigUpdate(ctx context.Context, server string, routeIndex int, realm, forwardHeader string) error {
	server = normalizeServer(server)
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	srv := getServer(cfg, server)
	if srv == nil || routeIndex < 0 || routeIndex >= len(srv.Routes) {
		return fmt.Errorf("路由不存在")
	}
	route := &srv.Routes[routeIndex]

	authIdx := findAuthIndex(route)
	if authIdx < 0 {
		return fmt.Errorf("该路由未配置 Basic Auth")
	}

	_, accounts, _ := pkgCaddy.BasicAuthFromHandler(route.Handle[authIdx])
	route.Handle[authIdx] = pkgCaddy.HandlerBasicAuth(realm, accounts)
	applyForwardHeader(route, forwardHeader)

	return s.client.ConfigLoad(ctx, cfg)
}

func findAuthIndex(route *pkgCaddy.Route) int {
	for i, h := range route.Handle {
		if _, _, ok := pkgCaddy.BasicAuthFromHandler(h); ok {
			return i
		}
	}
	return -1
}

// authUserIDVar 是 Caddy 认证成功后注入的用户名变量
const authUserIDVar = "{http.auth.user.id}"

// extractForwardHeader 从 handle 链中读取 forward-user 注入的 Header 名；未配置返回空串。
// 识别方式：handler == "headers" 且 request.set 中某个值包含 authUserIDVar。
func extractForwardHeader(handles []pkgCaddy.Handler) string {
	for _, h := range handles {
		if h["handler"] != "headers" {
			continue
		}
		req, _ := h["request"].(map[string]any)
		set, _ := req["set"].(map[string]any)
		for headerName, vals := range set {
			switch v := vals.(type) {
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok && s == authUserIDVar {
						return headerName
					}
				}
			case []string:
				for _, s := range v {
					if s == authUserIDVar {
						return headerName
					}
				}
			}
		}
	}
	return ""
}

// applyForwardHeader 维护 handle 链中的 forward-user headers handler：
//   - headerName 非空：确保链中存在该 handler（已存在则更新 header 名）
//   - headerName 为空：从链中移除该 handler
//
// handler 插入位置：authentication 之后、其余 handler 之前。
func applyForwardHeader(route *pkgCaddy.Route, headerName string) {
	// 移除已有的 forward-user handler（含 authUserIDVar 的 headers handler）
	filtered := route.Handle[:0]
	for _, h := range route.Handle {
		if h["handler"] == "headers" && extractForwardHeader([]pkgCaddy.Handler{h}) != "" {
			continue
		}
		filtered = append(filtered, h)
	}
	route.Handle = filtered

	if headerName == "" {
		return
	}

	newHandler := pkgCaddy.Handler{
		"handler": "headers",
		"request": map[string]any{
			"set": map[string]any{
				headerName: []string{authUserIDVar},
			},
		},
	}

	authIdx := findAuthIndex(route)
	if authIdx >= 0 {
		newHandles := make([]pkgCaddy.Handler, 0, len(route.Handle)+1)
		newHandles = append(newHandles, route.Handle[:authIdx+1]...)
		newHandles = append(newHandles, newHandler)
		newHandles = append(newHandles, route.Handle[authIdx+1:]...)
		route.Handle = newHandles
	} else {
		route.Handle = append([]pkgCaddy.Handler{newHandler}, route.Handle...)
	}
}
