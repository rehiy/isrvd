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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// 提取JWT令牌
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			c.Next()
			return
		}

		// 验证JWT令牌
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.Next()
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
