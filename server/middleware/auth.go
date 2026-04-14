package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"isrvd/server/config"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证令牌无效"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "代理 Header 缺失"})
			c.Abort()
			return
		}
		if _, exists := config.Members[username]; !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}
