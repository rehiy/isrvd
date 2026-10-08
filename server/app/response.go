package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxEditableJSONBytes int64 = 32 << 20

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
	if message == "" {
		message = "请求失败"
	}
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
	})
}

func respondBindError(c *gin.Context, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		respondError(c, http.StatusRequestEntityTooLarge, "文件超过在线编辑上限，请使用上传功能")
		return
	}
	respondError(c, http.StatusBadRequest, err.Error())
}

// respondResult 统一处理返回数据的 service 调用结果。
func respondResult(c *gin.Context, data any, err error) {
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", data)
}

// respondResultMsg 与 respondResult 相同，但成功时携带自定义提示文案；
// 无数据负载的操作类接口 data 传 nil。
func respondResultMsg(c *gin.Context, message string, data any, err error) {
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, message, data)
}
