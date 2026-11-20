package caddy

import (
	"context"
	"fmt"
	"slices"

	"github.com/rehiy/libgo/secure"

	"isrvd/pkgs/caddy"
)

// authUserIDVar 是 Caddy 认证成功后注入的用户名变量
const authUserIDVar = "{http.auth.user.id}"

// BasicAuthUser basic_auth 账号视图（密码不回显）
type BasicAuthUser struct {
	Username string `json:"username"` // basic_auth 登录用户名
}

// BasicAuthRouteView 带账号列表的路由视图
type BasicAuthRouteView struct {
	Index         int             `json:"index"`         // 路由在 revision 快照中的下标
	Revision      string          `json:"revision"`      // 所属服务全部路由的快照版本
	Name          string          `json:"name"`          // 路由名（仅展示）
	Realm         string          `json:"realm"`         // HTTP Basic realm
	ForwardHeader string          `json:"forwardHeader"` // 传递用户名的请求头；空表示未开启
	Users         []BasicAuthUser `json:"users"`         // 账号列表（不含密码 hash）
	Handlers      []caddy.Handler `json:"handlers"`      // 其余 handler 链（只读展示）
}

// BasicAuthSnapshot 包含认证路由与所属服务全部路由的同快照版本。
type BasicAuthSnapshot struct {
	Routes   []BasicAuthRouteView `json:"routes"`
	Revision string               `json:"revision"`
}

// BasicAuthConfigOptions 原子更新认证配置时的并发校验与账号移除选项。
type BasicAuthConfigOptions struct {
	Revision    string   `json:"revision"`
	RemoveUsers []string `json:"removeUsers"`
}

// BasicAuthList 保留默认列表响应。
func (s *Service) BasicAuthList(ctx context.Context, server string) ([]BasicAuthRouteView, error) {
	snapshot, err := s.BasicAuthSnapshot(ctx, server)
	if err != nil {
		return nil, err
	}
	return snapshot.Routes, nil
}

// BasicAuthSnapshot 从一次配置读取生成认证列表及全部路由的版本。
func (s *Service) BasicAuthSnapshot(ctx context.Context, server string) (*BasicAuthSnapshot, error) {
	server, err := normalizeServer(server)
	if err != nil {
		return nil, err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	srv := getServer(cfg, server)
	if srv == nil {
		return nil, ErrServerNotFound
	}
	revision, err := routeRevision(srv.Routes)
	if err != nil {
		return nil, err
	}
	var out []BasicAuthRouteView
	for i, route := range srv.Routes {
		for _, h := range route.Handle {
			realm, accounts, ok := caddy.BasicAuthFromHandler(h)
			if !ok {
				continue
			}
			users := make([]BasicAuthUser, len(accounts))
			for j, a := range accounts {
				users[j] = BasicAuthUser{Username: a.Username}
			}
			var others []caddy.Handler
			for _, hh := range route.Handle {
				if hh["handler"] != "authentication" {
					others = append(others, hh)
				}
			}
			out = append(out, BasicAuthRouteView{
				Index:         i,
				Revision:      revision,
				Name:          route.ID.String(),
				Realm:         realm,
				ForwardHeader: extractForwardHeader(route.Handle),
				Users:         users,
				Handlers:      others,
			})
			break
		}
	}
	return &BasicAuthSnapshot{Routes: out, Revision: revision}, nil
}

// BasicAuthUserCreate 向指定路由添加一个 basic_auth 账号
//
// forwardHeader 非空时，在 handle 链中维护一个 headers handler，将
// {http.auth.user.id} 注入到指定请求头中传给后端；空字符串表示移除该注入。
func (s *Service) BasicAuthUserCreate(ctx context.Context, server string, routeIndex int, username, password, realm, forwardHeader string, revision ...string) error {
	if username == "" || password == "" {
		return fmt.Errorf("用户名和密码不能为空")
	}
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	hash, err := secure.BcryptHash(password)
	if err != nil {
		return err
	}

	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if len(revision) > 0 && revision[0] != "" {
			if err := checkRouteRevision(srv.Routes, revision[0]); err != nil {
				return err
			}
		}
		route, err := getRoute(srv, routeIndex)
		if err != nil {
			return err
		}
		authIdx := findAuthIndex(route)

		if authIdx >= 0 {
			existingRealm, accounts, _ := caddy.BasicAuthFromHandler(route.Handle[authIdx])
			for _, account := range accounts {
				if account.Username == username {
					return fmt.Errorf("用户 %q 已存在", username)
				}
			}
			if realm == "" {
				realm = existingRealm
			}
			accounts = append(accounts, caddy.BasicAuthAccount{
				Username: username,
				Password: hash,
			})
			route.Handle[authIdx] = caddy.HandlerBasicAuth(realm, accounts)
		} else {
			newHandler := caddy.HandlerBasicAuth(realm, []caddy.BasicAuthAccount{
				{Username: username, Password: hash},
			})
			route.Handle = append([]caddy.Handler{newHandler}, route.Handle...)
		}

		applyForwardHeader(route, forwardHeader)
		return nil
	})
}

// BasicAuthUserDelete 从指定路由移除一个账号
//
// 若移除后账号列表为空，则同时删除 authentication handler 和 forward-user headers handler。
func (s *Service) BasicAuthUserDelete(ctx context.Context, server string, routeIndex int, username string, revision ...string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if len(revision) > 0 && revision[0] != "" {
			if err := checkRouteRevision(srv.Routes, revision[0]); err != nil {
				return err
			}
		}
		route, err := getRoute(srv, routeIndex)
		if err != nil {
			return err
		}

		authIdx := findAuthIndex(route)
		if authIdx < 0 {
			return fmt.Errorf("该路由未配置 Basic Auth")
		}

		realm, accounts, _ := caddy.BasicAuthFromHandler(route.Handle[authIdx])
		filtered := slices.DeleteFunc(accounts, func(account caddy.BasicAuthAccount) bool {
			return account.Username == username
		})
		if len(filtered) == len(accounts) {
			return fmt.Errorf("用户 %q 不存在", username)
		}

		if len(filtered) == 0 {
			route.Handle = slices.Delete(route.Handle, authIdx, authIdx+1)
			applyForwardHeader(route, "")
		} else {
			route.Handle[authIdx] = caddy.HandlerBasicAuth(realm, filtered)
		}
		return nil
	})
}

// BasicAuthConfigUpdate 原子更新认证配置和指定账号的移除，不允许清空最后账号。
func (s *Service) BasicAuthConfigUpdate(ctx context.Context, server string, routeIndex int, realm, forwardHeader string, options ...BasicAuthConfigOptions) error {
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}
	var option BasicAuthConfigOptions
	if len(options) > 0 {
		option = options[0]
	}
	return s.client.ConfigMutate(ctx, func(cfg *caddy.Config) error {
		srv := getServer(cfg, server)
		if srv == nil {
			return ErrServerNotFound
		}
		if option.Revision != "" || len(option.RemoveUsers) > 0 {
			if err := checkRouteRevision(srv.Routes, option.Revision); err != nil {
				return err
			}
		}
		route, err := getRoute(srv, routeIndex)
		if err != nil {
			return err
		}

		authIdx := findAuthIndex(route)
		if authIdx < 0 {
			return fmt.Errorf("该路由未配置 Basic Auth")
		}

		_, accounts, _ := caddy.BasicAuthFromHandler(route.Handle[authIdx])
		for _, username := range option.RemoveUsers {
			if !slices.ContainsFunc(accounts, func(account caddy.BasicAuthAccount) bool {
				return account.Username == username
			}) {
				return fmt.Errorf("用户 %q 不存在", username)
			}
		}
		accounts = slices.DeleteFunc(accounts, func(account caddy.BasicAuthAccount) bool {
			return slices.Contains(option.RemoveUsers, account.Username)
		})
		if len(accounts) == 0 {
			return fmt.Errorf("请至少保留一个账号；解除认证请使用账号删除操作")
		}
		route.Handle[authIdx] = caddy.HandlerBasicAuth(realm, accounts)
		applyForwardHeader(route, forwardHeader)
		return nil
	})
}

// ─── 辅助函数 ───

func findAuthIndex(route *caddy.Route) int {
	return slices.IndexFunc(route.Handle, func(h caddy.Handler) bool {
		_, _, ok := caddy.BasicAuthFromHandler(h)
		return ok
	})
}

// extractForwardHeader 从 handle 链中读取 forward-user 注入的 Header 名；未配置返回空串。
// 识别方式：handler == "headers" 且 request.set 中某个值包含 authUserIDVar。
func extractForwardHeader(handles []caddy.Handler) string {
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
				if slices.Contains(v, authUserIDVar) {
					return headerName
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
func applyForwardHeader(route *caddy.Route, headerName string) {
	// 移除已有的 forward-user handler（含 authUserIDVar 的 headers handler）
	route.Handle = slices.DeleteFunc(route.Handle, func(h caddy.Handler) bool {
		return h["handler"] == "headers" && extractForwardHeader([]caddy.Handler{h}) != ""
	})

	if headerName == "" {
		return
	}

	newHandler := caddy.Handler{
		"handler": "headers",
		"request": map[string]any{
			"set": map[string]any{
				headerName: []string{authUserIDVar},
			},
		},
	}

	authIdx := findAuthIndex(route)
	if authIdx >= 0 {
		newHandles := make([]caddy.Handler, 0, len(route.Handle)+1)
		newHandles = append(newHandles, route.Handle[:authIdx+1]...)
		newHandles = append(newHandles, newHandler)
		newHandles = append(newHandles, route.Handle[authIdx+1:]...)
		route.Handle = newHandles
	} else {
		route.Handle = append([]caddy.Handler{newHandler}, route.Handle...)
	}
}
