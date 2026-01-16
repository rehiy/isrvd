package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"isrvd/server/helper"
	"isrvd/server/model"
	"isrvd/server/service"
)

// 文件处理器
type FileHandler struct {
	fileService *service.FileService
}

// 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{
		fileService: service.GetFileService(),
	}
}

// 文件列表
func (h *FileHandler) List(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	files, err := h.fileService.List(abs)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "Directory not found")
		return
	}

	helper.RespondSuccess(c, "Files listed successfully", gin.H{
		"path":  req.Path,
		"files": files,
	})
}

// 删除文件
func (h *FileHandler) Delete(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.fileService.DeleteFile(abs)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}

	helper.RespondSuccess(c, "File deleted successfully", nil)
}

// 创建目录
func (h *FileHandler) Mkdir(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.fileService.Mkdir(abs)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create directory")
		return
	}

	helper.RespondSuccess(c, "Directory created successfully", nil)
}

// 新建文件
func (h *FileHandler) Create(c *gin.Context) {
	var req model.FileContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.fileService.Create(abs, req.Content)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}

	helper.RespondSuccess(c, "File created successfully", nil)
}

// 读取文件内容
func (h *FileHandler) Read(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	content, err := h.fileService.Read(abs)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}

	helper.RespondSuccess(c, "File content retrieved", gin.H{
		"path":    req.Path,
		"content": content,
	})
}

// 写入文件内容
func (h *FileHandler) Modify(c *gin.Context) {
	var req model.FileContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.fileService.Modify(abs, req.Content)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot save file")
		return
	}

	helper.RespondSuccess(c, "File saved successfully", nil)
}

// 重命名
func (h *FileHandler) Rename(c *gin.Context) {
	var req model.FileRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err := h.fileService.Rename(abs, req.Target)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}

	helper.RespondSuccess(c, "File renamed successfully", nil)
}

// 修改权限
func (h *FileHandler) Chmod(c *gin.Context) {
	var req model.FileChmodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid mode")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	err = h.fileService.Chmod(abs, os.FileMode(mode))
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot change permissions")
		return
	}

	helper.RespondSuccess(c, "Permissions changed successfully", nil)
}

// 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	path := c.PostForm("path")
	if path == "" {
		path = "/"
	}

	if !helper.ValidatePath(path) {
		helper.RespondError(c, http.StatusBadRequest, "Invalid path")
		return
	}

	abs := filepath.Join(helper.GetAbsolutePath(c, path), header.Filename)
	f, err := os.Create(abs)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot write file")
		return
	}

	helper.RespondSuccess(c, "File uploaded successfully", nil)
}

// 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	var req model.FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if !helper.ValidatePath(req.Path) {
		helper.RespondError(c, http.StatusBadRequest, "Invalid path")
		return
	}

	abs := helper.GetAbsolutePath(c, req.Path)
	f, err := os.Open(abs)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(req.Path))
	io.Copy(c.Writer, f)
}
