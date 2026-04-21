package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/helper"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

func (app *App) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Warn("Login request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	member, exists := config.Members[req.Username]
	if !exists || member.Password != req.Password {
		logman.Warn("Login failed", "username", req.Username)
		helper.RespondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		logman.Error("Token signing failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "token 生成失败")
		return
	}

	logman.Info("User logged in", "username", req.Username)
	helper.RespondSuccess(c, "Login successful", loginResponse{Token: tokenString, Username: req.Username})
}

func (app *App) logout(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		helper.RespondError(c, http.StatusUnauthorized, "未登录")
		return
	}
	logman.Info("User logged out", "username", username)
	helper.RespondSuccess(c, "Logout successful", nil)
}
