package middleware

import (
	"isrvd/server/helper"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"
)

// Recovery 中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logman.Error("Panic recovered", "error", err, "path", c.Request.URL.Path)
				helper.RespondError(c, http.StatusInternalServerError, "Internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
