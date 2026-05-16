// Package caddy 提供 Caddy Admin API 业务服务层
//
// 资源映射（业务约定）：
//   - 单 server 模式：默认 server 名为 srv0
//   - Route：servers/srv0/routes 数组项，对外用数组下标做主键
//   - 编辑器使用 RouteForm 简化模型，service 层负责与 Caddy 原生 Handler/Match 互转
package caddy

import (
	"fmt"
	"strings"

	"github.com/rehiy/libgo/logman"

	"isrvd/internal/registry"
	pkgcaddy "isrvd/pkgs/caddy"
)

// DefaultServerName 默认 server 名（业务约定，唯一一份）
const DefaultServerName = "srv0"

// HandlerKindReverseProxy 等：路由后端类型
const (
	HandlerKindReverseProxy = "reverse_proxy"
	HandlerKindFileServer   = "file_server"
	HandlerKindStaticResp   = "static_response"
	HandlerKindRaw          = "raw" // 透传原始 handle 数组
)

// CertSource 证书来源类型
const (
	CertSourceFile     = "file"     // load_files：磁盘文件路径
	CertSourcePEM      = "pem"      // load_pem：内联 PEM
	CertSourceAutomate = "automate" // automation.policies[].subjects：自动签发
)

// Service Caddy 业务服务
type Service struct {
	client *pkgcaddy.Client
}

// NewService 创建 Caddy 业务服务
func NewService() (*Service, error) {
	client := registry.CaddyClient
	if client == nil {
		logman.Error("Caddy client not initialized")
		return nil, fmt.Errorf("Caddy 未配置")
	}
	return &Service{client: client}, nil
}

// ─── 共享辅助函数 ───

func normalizeServer(name string) string {
	if name == "" {
		return DefaultServerName
	}
	return name
}

// getServer 取 server，不存在返回 nil
func getServer(cfg *pkgcaddy.Config, name string) *pkgcaddy.HTTPServer {
	if cfg == nil || cfg.Apps == nil || cfg.Apps.HTTP == nil {
		return nil
	}
	return cfg.Apps.HTTP.Servers[name]
}

// ensureServer 取 server，不存在则创建并初始化 listen
func ensureServer(cfg *pkgcaddy.Config, name string) *pkgcaddy.HTTPServer {
	if cfg.Apps == nil {
		cfg.Apps = &pkgcaddy.AppsConfig{}
	}
	if cfg.Apps.HTTP == nil {
		cfg.Apps.HTTP = &pkgcaddy.HTTPApp{}
	}
	if cfg.Apps.HTTP.Servers == nil {
		cfg.Apps.HTTP.Servers = map[string]*pkgcaddy.HTTPServer{}
	}
	srv, ok := cfg.Apps.HTTP.Servers[name]
	if !ok {
		srv = &pkgcaddy.HTTPServer{Listen: []string{":80"}}
		cfg.Apps.HTTP.Servers[name] = srv
	}
	return srv
}

func ensureTLS(cfg *pkgcaddy.Config) *pkgcaddy.TLSApp {
	if cfg.Apps == nil {
		cfg.Apps = &pkgcaddy.AppsConfig{}
	}
	if cfg.Apps.TLS == nil {
		cfg.Apps.TLS = &pkgcaddy.TLSApp{}
	}
	return cfg.Apps.TLS
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if v := strings.TrimSpace(s); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func toStrSlice(v any) []string {
	switch arr := v.(type) {
	case []string:
		return arr
	case []any:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// pickSecretStr 新值非空时用新值，否则保留旧值（与 system/config.go pickSecret 逻辑一致）
func pickSecretStr(newVal, oldVal string) string {
	if strings.TrimSpace(newVal) != "" {
		return newVal
	}
	return oldVal
}
