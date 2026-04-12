package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"isrvd/server/config"
)

// JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 内网代理 Header 认证模式（启用后完全替代 JWT，不回退）
		if config.ProxyHeaderName != "" {
			username := c.GetHeader(config.ProxyHeaderName)
			if username == "" {
				c.JSON(403, gin.H{"error": "代理 Header 缺失"})
				c.Abort()
				return
			}
			if _, exists := config.Members[username]; !exists {
				c.JSON(403, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}
			c.Set("username", username)
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// WebSocket 连接时才允许从 query 参数获取 token
		if tokenStr == "" && c.GetHeader("Upgrade") == "websocket" {
			tokenStr = c.Query("token")
		}

		// 如果没有获取到 token，则拒绝请求
		if tokenStr == "" {
			c.JSON(401, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		// 验证JWT令牌
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "认证令牌无效"})
			c.Abort()
			return
		}

		// 解析JWT声明
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
