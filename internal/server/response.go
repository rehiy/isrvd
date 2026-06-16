package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 通用API响应结构
type APIResponse struct {
	Success bool   `json:"success"`           // 请求是否成功
	Message string `json:"message,omitempty"` // 提示信息
	Payload any    `json:"payload,omitempty"` // 响应数据负载
}

// respondSuccess 返回成功响应
func respondSuccess(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Payload: data,
	})
}

// respondError 返回错误响应
func respondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
	})
}
