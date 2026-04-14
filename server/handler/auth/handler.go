package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.GetAuthService(),
	}
}

// Login 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Warn("Login request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	result, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		logman.Warn("Login failed", "username", req.Username, "error", err)
		helper.RespondError(c, http.StatusUnauthorized, err.Error())
		return
	}

	logman.Info("User logged in", "username", req.Username)
	helper.RespondSuccess(c, "Login successful", LoginResponse{
		Token:    result.Token,
		Username: result.Username,
	})
}

// Logout 登出处理
func (h *AuthHandler) Logout(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		logman.Warn("Logout attempt without authentication")
		helper.RespondError(c, http.StatusUnauthorized, "未登录")
		return
	}

	logman.Info("User logged out", "username", username)
	helper.RespondSuccess(c, "Logout successful", nil)
}
