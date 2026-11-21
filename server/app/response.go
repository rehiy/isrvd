package app

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/helper"
	"isrvd/server/i18n"
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
		Message: i18n.T(c, message),
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
		Message: i18n.T(c, message),
	})
}

func respondBindError(c *gin.Context, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		respondError(c, http.StatusRequestEntityTooLarge, "文件超过在线编辑上限，请使用上传功能")
		return
	}
	respondError(c, http.StatusBadRequest, err.Error())
}

// respondEditableReadError 处理各文件后端共用的在线编辑错误。
func respondEditableReadError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, helper.ErrTooLarge):
		respondError(c, http.StatusRequestEntityTooLarge, "文件超过在线编辑上限，请使用下载功能")
	case errors.Is(err, helper.ErrBinary):
		respondError(c, http.StatusUnsupportedMediaType, "文件不是文本文件，无法在线编辑")
	default:
		return false
	}
	return true
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

// respondFileDownload 只在尚未提交响应时返回 JSON。部分传输保留长度，
// 由 HTTP 客户端检测响应体中断。
func respondFileDownload(c *gin.Context, name string, src io.ReadCloser, size int64, errorStatus int) {
	defer src.Close()
	if name != "" {
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	if err := copyDownload(c.Writer, src, size); err != nil {
		if c.Writer.Written() {
			logman.Warn("file download interrupted", "error", err)
			return
		}
		c.Header("Content-Length", "")
		c.Header("Content-Type", "")
		c.Header("Content-Disposition", "")
		respondError(c, errorStatus, err.Error())
	}
}

func copyDownload(dst io.Writer, src io.Reader, size int64) error {
	if size == 0 {
		return nil
	}
	if stream, ok := src.(io.WriterTo); ok {
		// 保留 SFTP 的并发读取，同时遵守响应已经声明的长度。
		bounded := &downloadWriter{dst: dst, remaining: size}
		_, err := stream.WriteTo(bounded)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if bounded.remaining != 0 {
			return io.ErrUnexpectedEOF
		}
		return nil
	}
	_, err := io.CopyN(dst, src, size)
	return err
}

type downloadWriter struct {
	dst       io.Writer
	remaining int64
}

func (w *downloadWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.EOF
	}
	n, err := w.dst.Write(p[:min(int64(len(p)), w.remaining)])
	w.remaining -= int64(n)
	if err == nil && n < len(p) {
		err = io.EOF
	}
	return n, err
}
