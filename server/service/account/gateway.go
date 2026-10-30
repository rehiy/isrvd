package account

import (
	"net"
	"net/http"

	"isrvd/pkgs/helper"
	"isrvd/server/config"
)

// 中控网关与账户体系的身份约定：网关在入口处校验调用者，把结果写入内部身份头交给同进程的 isrvd，
// 外部传入的同名头一律丢弃。与此相关的全部知识都收在本文件。

// GatewayUserHeader 网关写给同进程 isrvd 的内部身份头
const GatewayUserHeader = "X-Isrvd-Gateway-User"

// TrustGatewayIdentity 在网关入口处清理并重建内部身份头：外部传入的一律丢弃；
// 启用代理 Header 登录时校验来源，把结果写入内部身份头，原始认证头不再透传。
func TrustGatewayIdentity(r *http.Request) {
	snapshot := config.Current()
	r.Header.Del(GatewayUserHeader)
	if snapshot.THA.Enabled {
		user, _ := HeaderAuth(snapshot, r)
		r.Header.Del(snapshot.THA.HeaderName)
		if user != "" {
			r.Header.Set(GatewayUserHeader, user)
		}
	}
}

// GatewayCredentialHeaders 返回携带用户凭据、不得下发给受管机的请求头。
func GatewayCredentialHeaders() []string {
	return []string{"Authorization", "Cookie", GatewayUserHeader, config.Current().THA.HeaderName}
}

// GatewayUser 认证已由 TrustGatewayIdentity 清理过的请求，返回用户名；认证失败或成员不存在返回空串。
func (s *Service) GatewayUser(r *http.Request, allowQueryToken bool) string {
	snapshot := config.Current()
	username := r.Header.Get(GatewayUserHeader)
	if !snapshot.THA.Enabled {
		username, _ = s.jwtCheck(snapshot, r, allowQueryToken)
	}
	if username == "" || snapshot.Members[username] == nil {
		return ""
	}
	return username
}

func isLoopback(r *http.Request) bool {
	ip := net.ParseIP(helper.RemoteHost(r))
	return ip != nil && ip.IsLoopback()
}
