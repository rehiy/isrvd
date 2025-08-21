package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"isrvd/internal/models"
	"isrvd/internal/services"
	"isrvd/pkg/utils"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileService *services.FileService
}

// NewFileHandler 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{
		fileService: services.FileServiceInstance,
	}
}

// ListFiles 文件列表
func (h *FileHandler) ListFiles(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	files, err := h.fileService.ListFiles(path)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "Directory not found")
		return
	}

	utils.RespondSuccess(c, "Files listed successfully", gin.H{
		"path":  path,
		"files": files,
	})
}

// Upload 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	path := c.PostForm("path")
	if path == "" {
		path = "/"
	}

	if !utils.ValidatePath(path) {
		utils.RespondError(c, http.StatusBadRequest, "Invalid path")
		return
	}

	absPath := filepath.Join(utils.GetAbsolutePath(path), header.Filename)
	f, err := os.Create(absPath)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot write file")
		return
	}

	utils.RespondSuccess(c, "File uploaded successfully", nil)
}

// Download 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	path := c.Query("file")
	if path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	if !utils.ValidatePath(path) {
		utils.RespondError(c, http.StatusBadRequest, "Invalid path")
		return
	}

	absPath := utils.GetAbsolutePath(path)
	f, err := os.Open(absPath)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(absPath))
	io.Copy(c.Writer, f)
}

// Delete 删除文件
func (h *FileHandler) Delete(c *gin.Context) {
	path := c.Query("file")
	if path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	err := h.fileService.DeleteFile(path)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}

	utils.RespondSuccess(c, "File deleted successfully", nil)
}

// CreateDirectory 创建目录
func (h *FileHandler) CreateDirectory(c *gin.Context) {
	var req models.MkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.fileService.CreateDirectory(req.Path, req.Name)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot create directory")
		return
	}

	utils.RespondSuccess(c, "Directory created successfully", nil)
}

// CreateFile 新建文件
func (h *FileHandler) CreateFile(c *gin.Context) {
	var req models.NewFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.fileService.CreateFile(req.Path, req.Name, req.Content)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}

	utils.RespondSuccess(c, "File created successfully", nil)
}

// EditFile 编辑文件
func (h *FileHandler) EditFile(c *gin.Context) {
	path := c.Query("file")
	if path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	// 读取文件内容
	if c.Request.Method == "GET" {
		content, err := h.fileService.ReadFile(path)
		if err != nil {
			utils.RespondError(c, http.StatusNotFound, "File not found")
			return
		}

		utils.RespondSuccess(c, "File content retrieved", gin.H{
			"path":    path,
			"content": content,
		})
		return
	}

	// 保存文件内容
	if c.Request.Method == "PUT" {
		var req models.EditFileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
			return
		}

		err := h.fileService.WriteFile(path, req.Content)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "Cannot save file")
			return
		}

		utils.RespondSuccess(c, "File saved successfully", nil)
		return
	}

	utils.RespondError(c, http.StatusMethodNotAllowed, "Method not allowed")
}

// Rename 重命名
func (h *FileHandler) Rename(c *gin.Context) {
	var req models.RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.fileService.RenameFile(req.OldPath, req.NewName)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}

	utils.RespondSuccess(c, "File renamed successfully", nil)
}

// ChangeMode 修改权限
func (h *FileHandler) ChangeMode(c *gin.Context) {
	path := c.Query("file")
	if path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	// 获取当前权限
	if c.Request.Method == "GET" {
		info, err := h.fileService.GetFileInfo(path)
		if err != nil {
			utils.RespondError(c, http.StatusNotFound, "File not found")
			return
		}

		utils.RespondSuccess(c, "File mode retrieved", gin.H{
			"file": filepath.Base(path),
			"mode": strconv.FormatUint(uint64(info.Mode().Perm()), 8),
		})
		return
	}

	// 修改权限
	if c.Request.Method == "PUT" {
		var req models.ChmodRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
			return
		}

		mode, err := strconv.ParseUint(req.Mode, 8, 32)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "Invalid mode")
			return
		}

		err = h.fileService.ChangeMode(path, os.FileMode(mode))
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "Cannot change permissions")
			return
		}

		utils.RespondSuccess(c, "Permissions changed successfully", nil)
		return
	}

	utils.RespondError(c, http.StatusMethodNotAllowed, "Method not allowed")
}
