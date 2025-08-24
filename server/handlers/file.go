package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"isrvd/server/helpers/utils"
	"isrvd/server/models"
	"isrvd/server/services"
)

// 文件处理器
type FileHandler struct {
	fileService *services.FileService
}

// 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{
		fileService: services.NewFileService(),
	}
}

// 文件列表
func (h *FileHandler) ListFiles(c *gin.Context) {
	var req models.ListFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	files, err := h.fileService.ListFiles(req.Path)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "Directory not found")
		return
	}

	utils.RespondSuccess(c, "Files listed successfully", gin.H{
		"path":  req.Path,
		"files": files,
	})
}

// 删除文件
func (h *FileHandler) Delete(c *gin.Context) {
	var req models.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	err := h.fileService.DeleteFile(req.Path)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}

	utils.RespondSuccess(c, "File deleted successfully", nil)
}

// 创建目录
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

// 新建文件
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

// 读取文件内容
func (h *FileHandler) ReadFile(c *gin.Context) {
	var req models.ReadFileHandlerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	content, err := h.fileService.ReadFile(req.Path)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "File not found")
		return
	}

	utils.RespondSuccess(c, "File content retrieved", gin.H{
		"path":    req.Path,
		"content": content,
	})
}

// 写入文件内容
func (h *FileHandler) WriteFile(c *gin.Context) {
	var req models.WriteFileHandlerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}
	if req.Content == "" {
		utils.RespondError(c, http.StatusBadRequest, "No content provided")
		return
	}

	err := h.fileService.WriteFile(req.Path, req.Content)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot save file")
		return
	}

	utils.RespondSuccess(c, "File saved successfully", nil)
}

// 重命名
func (h *FileHandler) Rename(c *gin.Context) {
	var req models.RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.fileService.RenameFile(req.Path, req.NewPath)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}

	utils.RespondSuccess(c, "File renamed successfully", nil)
}

// 修改权限
func (h *FileHandler) ChangeMode(c *gin.Context) {
	var req models.ChmodHandlerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	// 如果没有提供 mode，则获取当前权限
	if req.Mode == "" {
		info, err := h.fileService.GetFileInfo(req.Path)
		if err != nil {
			utils.RespondError(c, http.StatusNotFound, "File not found")
			return
		}

		utils.RespondSuccess(c, "File mode retrieved", gin.H{
			"file": filepath.Base(req.Path),
			"mode": strconv.FormatUint(uint64(info.Mode().Perm()), 8),
		})
		return
	}

	// 如果提供了 mode，则修改权限
	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid mode")
		return
	}

	err = h.fileService.ChangeMode(req.Path, os.FileMode(mode))
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Cannot change permissions")
		return
	}

	utils.RespondSuccess(c, "Permissions changed successfully", nil)
}

// 上传文件
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

// 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	var req models.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Path == "" {
		utils.RespondError(c, http.StatusBadRequest, "No file specified")
		return
	}

	if !utils.ValidatePath(req.Path) {
		utils.RespondError(c, http.StatusBadRequest, "Invalid path")
		return
	}

	f, err := os.Open(utils.GetAbsolutePath(req.Path))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(req.Path))
	io.Copy(c.Writer, f)
}
