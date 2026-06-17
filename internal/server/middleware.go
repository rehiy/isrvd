package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	svcAccount "isrvd/internal/service/account"
	svcSystem "isrvd/internal/service/system"
)

// AuthMiddleware 认证中间件
// - AccessAnon 路由：可选认证，失败时放行
// - 其他路由：强制认证，失败时返回 401
func AuthMiddleware(routeIndex map[string]Route, svc *svcAccount.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if route, ok := matchRoute(routeIndex, c.Request.Method, c.FullPath()); ok {
			if route.QueryToken {
				c.Set("routeQueryToken", true)
			}
			if route.Access == AccessAnon {
				if username := svc.AuthMix(c); username != "" {
					c.Set("username", username)
				}
				c.Next()
				return
			}
		}

		username, errMsg := svc.Auth(c)
		if username == "" {
			respondError(c, http.StatusUnauthorized, errMsg)
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}

// PermMiddleware 权限验证中间件
// 基于 METHOD+PATH 进行集中式权限校验
func PermMiddleware(routeIndex map[string]Route, svc *svcAccount.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		route, ok := matchRoute(routeIndex, c.Request.Method, path)
		if path == "" || !ok {
			respondError(c, http.StatusForbidden, "未授权的访问路径")
			c.Abort()
			return
		}

		// 防御性校验：非匿名路由必须已认证。
		// 正常情况下 AuthMiddleware 已保证此处 username 非空（否则已返回 401），
		// 此处作为兜底断言，避免 AuthMiddleware 被误改或绕过时越权放行。
		if route.Access != AccessAnon && c.GetString("username") == "" {
			respondError(c, http.StatusForbidden, "请先登录")
			c.Abort()
			return
		}

		// 仅 AccessPerm 且声明了 Module 的路由才执行细粒度权限校验；
		// AccessAuth 路由登录即可访问，无需 PermCheck。
		if route.Access == AccessPerm && route.Module != "" {
			if err := svc.PermCheck(c.GetString("username"), route.Label, c.Request.Method, path); err != nil {
				respondError(c, http.StatusForbidden, err.Error())
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AuditMiddleware 操作审计中间件
// 根据路由 Audit 策略决定是否记录：0 按 Method，<0 忽略，>0 强制记录。
func AuditMiddleware(routeIndex map[string]Route, svc *svcSystem.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		route, _ := matchRoute(routeIndex, c.Request.Method, c.FullPath())
		isWS := strings.EqualFold(c.GetHeader("Upgrade"), "websocket")
		if route.Audit < AuditByMethod || (route.Audit == AuditByMethod && !isWS && c.Request.Method == http.MethodGet) {
			c.Next()
			return
		}

		startTime := time.Now()
		var body string
		if !isWS {
			body = svc.BodyRead(c)
		}

		c.Next()
		svc.AuditRecord(c, startTime, body)
	}
}

// securityHeadersMiddleware 安全响应头中间件
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")
		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP：限制资源加载和脚本执行，降低 XSS 风险
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")

		c.Next()
	}
}

// ─── 辅助函数 ───

func matchRoute(routeIndex map[string]Route, method, path string) (Route, bool) {
	route, ok := routeIndex[method+" "+path]
	if !ok {
		route, ok = routeIndex["ANY "+path]
	}
	return route, ok
}
