package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/model"
	"isrvd/server/service"
)

// zip处理器
type ZipHandler struct {
	zipService *service.ZipService
}

// 创建zip处理器
func NewZipHandler() *ZipHandler {
	return &ZipHandler{
		zipService: service.GetZipService(),
	}
}

// 创建压缩文件
func (h *ZipHandler) Zip(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.zipService.Zip(abs)
	if err != nil {
		logman.Error("Create zip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create zip archive")
		return
	}

	logman.Info("Zip created", "path", abs)
	helper.RespondSuccess(c, "Archive created successfully", nil)
}

// 解压文件
func (h *ZipHandler) Unzip(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.zipService.Unzip(abs)
	if err != nil {
		logman.Error("Unzip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot extract archive")
		return
	}

	logman.Info("Archive extracted", "path", abs)
	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}
