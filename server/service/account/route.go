package account

import "strings"

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
