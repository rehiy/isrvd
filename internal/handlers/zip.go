package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/models"
	"isrvd/internal/services"
	"isrvd/pkg/utils"
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
	var req models.GetZipInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	files, err := h.zipService.GetZipInfo(req.Path)
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
	var req models.IsZipFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "Path is required")
		return
	}

	isZip := h.zipService.IsZipFile(req.Path)
	utils.RespondSuccess(c, "File type checked successfully", gin.H{
		"isZip": isZip,
	})
}
