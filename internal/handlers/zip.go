package handlers

import (
	"net/http"

	"filer/internal/models"
	"filer/internal/services"
	"filer/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ZipHandler zip处理器
type ZipHandler struct {
	zipService *services.ZipService
}

// NewZipHandler 创建zip处理器
func NewZipHandler() *ZipHandler {
	return &ZipHandler{
		zipService: services.ZipServiceInstance,
	}
}

// CreateZip 创建压缩文件
func (h *ZipHandler) CreateZip(c *gin.Context) {
	var req models.ZipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.zipService.CreateZip(req.Path, req.ZipName)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot create zip archive")
		return
	}

	utils.RespondSuccess(c, "Archive created successfully", nil)
}

// ExtractZip 解压文件
func (h *ZipHandler) ExtractZip(c *gin.Context) {
	var req models.UnzipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.zipService.ExtractZip(req.Path, req.ZipName)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot extract archive")
		return
	}

	utils.RespondSuccess(c, "Archive extracted successfully", nil)
}

// GetZipInfo 获取zip文件信息
func (h *ZipHandler) GetZipInfo(c *gin.Context) {
	zipPath := c.Query("path")
	if zipPath == "" {
		utils.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	files, err := h.zipService.GetZipInfo(zipPath)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot read zip file")
		return
	}

	utils.RespondSuccess(c, "Zip info retrieved successfully", gin.H{
		"files": files,
	})
}

// IsZipFile 判断文件是否为zip文件
func (h *ZipHandler) IsZipFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		utils.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	isZip := h.zipService.IsZipFile(filePath)
	utils.RespondSuccess(c, "File type checked successfully", gin.H{
		"isZip": isZip,
	})
}
