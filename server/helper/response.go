package helper

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"isrvd/server/config"
)

// 通用API响应结构
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

// 返回成功响应
func RespondSuccess(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Payload: data,
	})
}

// 返回错误响应
func RespondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
	})
}

// 获取绝对路径
func GetAbsolutePath(path string) string {
	cfg := config.GetGlobal()
	return filepath.Join(cfg.BaseDir, filepath.Clean(path))
}

// 验证路径安全性，防止目录遍历攻击
func ValidatePath(path string) bool {
	cleanPath := filepath.Clean(path)
	return !strings.Contains(cleanPath, "..")
}

// 从请求中获取令牌
func GetTokenFromRequest(c *gin.Context) string {
	// 优先从 Authorization 头获取
	token := c.GetHeader("Authorization")
	if token != "" {
		// 移除 "Bearer " 前缀（如果存在）
		token = strings.TrimPrefix(token, "Bearer ")
		return token
	}

	// 从查询参数获取
	return c.Query("token")
}
