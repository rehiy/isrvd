package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"isrvd/config"
	"isrvd/internal/helper"
)

// 认证中间件工厂
func AuthMiddleware() gin.HandlerFunc {
	if config.ProxyHeaderName == "" {
		return JwtAuthMiddleware()
	}
	return ProxyHeaderAuthMiddleware()
}

// JWT 认证中间件
func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// WebSocket 连接时允许从 query 参数获取 token
		if tokenStr == "" && c.GetHeader("Upgrade") == "websocket" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			helper.RespondError(c, http.StatusUnauthorized, "未提供认证令牌")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			helper.RespondError(c, http.StatusUnauthorized, "认证令牌无效")
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, exists := claims["sub"].(string); exists {
				if _, memberExists := config.Members[sub]; memberExists {
					c.Set("username", sub)
				}
			}
		}
		c.Next()
	}
}

// 内网代理 Header 认证中间件
// 启用条件：config.ProxyHeaderName 非空
// Header 缺失或用户不存在时返回 403，不回退到 JWT
func ProxyHeaderAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetHeader(config.ProxyHeaderName)
		if username == "" {
			helper.RespondError(c, http.StatusForbidden, "代理 Header 缺失")
			c.Abort()
			return
		}
		if _, exists := config.Members[username]; !exists {
			helper.RespondError(c, http.StatusForbidden, "用户不存在")
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}

// moduleLabel 模块显示名映射，统一错误提示中的大小写与空格
var moduleLabel = map[string]string{
	"filer":   "文件管理",
	"docker":  "Docker",
	"swarm":   "Swarm",
	"compose": "Compose",
	"apisix":  "APISIX",
	"agent":   "AI Agent",
	"system":  "系统管理",
}

// PermMiddleware 模块权限检查中间件
// module: 模块名（filer/docker/swarm/compose/apisix/agent/system）
// write: true 表示需要写权限（rw），false 表示只需读权限（r 或 rw）
// 主账号（PrimaryMember）始终放行
func PermMiddleware(module string, write bool) gin.HandlerFunc {
	label := moduleLabel[module]
	if label == "" {
		label = module
	}
	return func(c *gin.Context) {
		username := c.GetString("username")
		// 主账号始终拥有全部权限
		if username == config.PrimaryMember {
			c.Next()
			return
		}
		member, exists := config.Members[username]
		if !exists {
			helper.RespondError(c, http.StatusForbidden, "用户不存在")
			c.Abort()
			return
		}
		perm := member.Permissions[module]
		if write {
			if perm != "rw" {
				helper.RespondError(c, http.StatusForbidden, "无 "+label+" 模块写权限")
				c.Abort()
				return
			}
		} else {
			if perm != "r" && perm != "rw" {
				helper.RespondError(c, http.StatusForbidden, "无 "+label+" 模块访问权限")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
