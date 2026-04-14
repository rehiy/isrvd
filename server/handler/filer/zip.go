package filer

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/service"
)

// ZipHandler zip处理器
type ZipHandler struct {
	zipService *service.ZipService
}

// NewZipHandler 创建zip处理器
func NewZipHandler() *ZipHandler {
	return &ZipHandler{
		zipService: service.GetZipService(),
	}
}

// Zip 创建压缩文件
func (h *ZipHandler) Zip(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Zip request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	abs := getAbsolutePath(c, req.Path)
	err := h.zipService.Zip(abs)
	if err != nil {
		logman.Error("Create zip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法创建压缩文件")
		return
	}

	logman.Info("Zip created", "path", abs)
	helper.RespondSuccess(c, "Archive created successfully", nil)
}

// Unzip 解压文件
func (h *ZipHandler) Unzip(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Unzip request invalid", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	abs := getAbsolutePath(c, req.Path)
	err := h.zipService.Unzip(abs)
	if err != nil {
		logman.Error("Unzip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法解压文件")
		return
	}

	logman.Info("Archive extracted", "path", abs)
	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}
