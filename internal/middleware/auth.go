package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/pkg/auth"
	"isrvd/pkg/utils"
)

// Auth 认证中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := utils.GetTokenFromRequest(c)
		if token == "" || !auth.Manager.ValidateToken(token) {
			utils.RespondError(c, http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}
