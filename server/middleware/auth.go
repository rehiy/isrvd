package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/helper"
	"isrvd/server/service"
)

// 认证中间件
func Auth() gin.HandlerFunc {
	var session = service.NewAuthService()

	return func(c *gin.Context) {
		token := helper.GetTokenFromRequest(c)
		if token == "" || !session.ValidateToken(token) {
			helper.RespondError(c, http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}
