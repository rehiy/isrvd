package filer

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/pkgs/archive"
	"isrvd/server/helper"
)

// ZipHandler zip处理器
type ZipHandler struct {
	zipper *archive.Zipper
}

// NewZipHandler 创建zip处理器
func NewZipHandler() *ZipHandler {
	return &ZipHandler{
		zipper: archive.NewZipper(),
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
	err := h.zipper.Zip(abs)
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
	err := h.zipper.Unzip(abs)
	if err != nil {
		logman.Error("Unzip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法解压文件")
		return
	}

	logman.Info("Archive extracted", "path", abs)
	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}
