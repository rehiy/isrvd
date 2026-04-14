package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rehiy/pango/logman"

	"isrvd/server/config"
	"isrvd/server/helper"
)

// AuthHandler 认证处理器
type AuthHandler struct{}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// Login 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Warn("Login request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证用户凭据
	member, exists := config.Members[req.Username]
	if !exists || member.Password != req.Password {
		logman.Warn("Login failed", "username", req.Username)
		helper.RespondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	// 签名 token
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		logman.Error("Token signing failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "token 生成失败")
		return
	}

	logman.Info("User logged in", "username", req.Username)
	helper.RespondSuccess(c, "Login successful", LoginResponse{
		Token:    tokenString,
		Username: req.Username,
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
