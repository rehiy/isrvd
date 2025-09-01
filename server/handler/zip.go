package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

	err := h.zipService.Zip(req.Path)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create zip archive")
		return
	}

	helper.RespondSuccess(c, "Archive created successfully", nil)
}

// 解压文件
func (h *ZipHandler) Unzip(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.zipService.Unzip(req.Path)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot extract archive")
		return
	}

	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}
