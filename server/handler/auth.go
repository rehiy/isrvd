package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/model"
	"isrvd/server/service"
)

// 认证处理器
type AuthHandler struct {
	authService *service.AuthService
}

// 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.GetAuthService(),
	}
}

// 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Warn("Login request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		logman.Warn("Login failed", "username", req.Username, "error", err)
		helper.RespondError(c, http.StatusUnauthorized, err.Error())
		return
	}

	logman.Info("User logged in", "username", req.Username)
	helper.RespondSuccess(c, "Login successful", resp)
}

// 登出处理
func (h *AuthHandler) Logout(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		logman.Warn("Logout attempt without authentication")
		helper.RespondError(c, http.StatusUnauthorized, "Not logged in")
		return
	}

	logman.Info("User logged out", "username", username)
	helper.RespondSuccess(c, "Logout successful", nil)
}
