package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
		authService: service.NewAuthService(),
	}
}

// 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		helper.RespondError(c, http.StatusUnauthorized, err.Error())
		return
	}

	helper.RespondSuccess(c, "Login successful", resp)
}

// 登出处理
func (h *AuthHandler) Logout(c *gin.Context) {
	token := helper.GetTokenFromRequest(c)
	h.authService.Logout(token)
	helper.RespondSuccess(c, "Logged out successfully", nil)
}
