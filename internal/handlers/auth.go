package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/models"
	"isrvd/internal/services"
	"isrvd/pkg/utils"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: services.AuthServiceInstance,
	}
}

// Login 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.RespondSuccess(c, "Login successful", resp)
}

// Logout 登出处理
func (h *AuthHandler) Logout(c *gin.Context) {
	token := utils.GetTokenFromRequest(c)
	h.authService.Logout(token)
	utils.RespondSuccess(c, "Logged out successfully", nil)
}
