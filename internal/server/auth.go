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
	return HeaderAuthMiddleware()
}

// MixAuthMiddleware 可选认证中间件
// 认证成功时写入 username，失败时直接放行（不中断请求）
// 认证模式在工厂函数调用时确定，避免每次请求重复判断静态配置
func MixAuthMiddleware() gin.HandlerFunc {
	if config.ProxyHeaderName != "" {
		// Header 认证模式
		return func(c *gin.Context) {
			if username := extractHeaderUsername(c); username != "" {
				c.Set("username", username)
			}
			c.Next()
		}
	}
	// JWT 认证模式
	return func(c *gin.Context) {
		if username := extractJwtUsername(c); username != "" {
			c.Set("username", username)
		}
		c.Next()
	}
}

// JWT 认证中间件
func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := extractJwtUsername(c)
		if username == "" {
			// 区分"未提供 token"与"token 无效"两种情况给出不同提示
			authHeader := c.GetHeader("Authorization")
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == "" && c.GetHeader("Upgrade") != "websocket" {
				helper.RespondError(c, http.StatusUnauthorized, "未提供认证令牌")
			} else {
				helper.RespondError(c, http.StatusUnauthorized, "认证令牌无效")
			}
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}

// 内网代理 Header 认证中间件
// 启用条件：config.ProxyHeaderName 非空
// Header 缺失或用户不存在时返回 403，不回退到 JWT
func HeaderAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := extractHeaderUsername(c)
		if username == "" {
			if c.GetHeader(config.ProxyHeaderName) == "" {
				helper.RespondError(c, http.StatusForbidden, "代理 Header 缺失")
			} else {
				helper.RespondError(c, http.StatusForbidden, "用户不存在")
			}
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
	"agent":   "AI Agent",
	"apisix":  "APISIX",
	"docker":  "Docker",
	"swarm":   "Swarm",
	"compose": "Compose",
	"system":  "系统管理",
	"shell":   "Shell终端",
}

// PermMiddleware 模块权限检查中间件
// module: 模块名（filer/agent/apisix/docker/swarm/compose/system/shell）
// write: true 表示需要写权限（rw），false 表示只需读权限（r 或 rw）
func PermMiddleware(module string, write bool) gin.HandlerFunc {
	label := moduleLabel[module]
	if label == "" {
		label = module
	}
	return func(c *gin.Context) {
		username := c.GetString("username")
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

// extractJwtUsername 从 Authorization Header（或 WebSocket query）中解析 JWT，
// 返回有效且存在于成员列表中的用户名；否则返回空字符串
func extractJwtUsername(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	// WebSocket 连接时允许从 query 参数获取 token
	if tokenStr == "" && c.GetHeader("Upgrade") == "websocket" {
		tokenStr = c.Query("token")
	}
	if tokenStr == "" {
		return ""
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}
	sub, _ := claims["sub"].(string)
	if _, exists := config.Members[sub]; !exists {
		return ""
	}
	return sub
}

// extractHeaderUsername 从代理 Header 中读取用户名，
// 返回存在于成员列表中的用户名；否则返回空字符串
func extractHeaderUsername(c *gin.Context) string {
	username := c.GetHeader(config.ProxyHeaderName)
	if username == "" {
		return ""
	}
	if _, exists := config.Members[username]; !exists {
		return ""
	}
	return username
}
