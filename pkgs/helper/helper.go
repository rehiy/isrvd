// Package helper 提供与业务无关的通用小工具（HTTP 请求判断、路由模式转换、在线编辑限长缓冲），
// 供网关、应用与各服务共用，保证各处判断一致。
package helper

import (
	"net"
	"net/http"
	"strings"
)

// RemoteHost 只取连接对端 IP（不含端口），不信任客户端提供的转发头。
func RemoteHost(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// IsWebSocket 判断请求是否为 WebSocket 升级（Upgrade 头大小写不敏感）。
func IsWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// MuxPattern 把 Gin 风格的路由（:param、*wildcard）转成 http.ServeMux 的匹配模式。
// method 为空或 ANY 时只返回路径部分。中控网关与应用的权限索引共用它，保证两边匹配规则一致。
func MuxPattern(method, path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		switch {
		case strings.HasPrefix(part, ":"):
			parts[i] = "{" + part[1:] + "}"
		case strings.HasPrefix(part, "*"):
			parts[i] = "{" + part[1:] + "...}"
		}
	}
	pattern := strings.Join(parts, "/")
	if method == "" || method == "ANY" {
		return pattern
	}
	return method + " " + pattern
}
