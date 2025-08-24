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
		zipService: service.NewZipService(),
	}
}

// 创建压缩文件
func (h *ZipHandler) CreateZip(c *gin.Context) {
	var req model.ZipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.zipService.CreateZip(req.Path, req.ZipName)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create zip archive")
		return
	}

	helper.RespondSuccess(c, "Archive created successfully", nil)
}

// 解压文件
func (h *ZipHandler) ExtractZip(c *gin.Context) {
	var req model.UnzipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.zipService.ExtractZip(req.Path, req.ZipName)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot extract archive")
		return
	}

	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}

// 获取zip文件信息
func (h *ZipHandler) GetZipInfo(c *gin.Context) {
	var req model.GetZipInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		helper.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	files, err := h.zipService.GetZipInfo(req.Path)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot read zip file")
		return
	}

	helper.RespondSuccess(c, "Zip info retrieved successfully", gin.H{
		"files": files,
	})
}

// 判断文件是否为zip文件
func (h *ZipHandler) IsZipFile(c *gin.Context) {
	var req model.IsZipFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		helper.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	isZip := h.zipService.IsZipFile(req.Path)
	helper.RespondSuccess(c, "File type checked successfully", gin.H{
		"isZip": isZip,
	})
}
