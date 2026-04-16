package filer

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/filer"
	"github.com/rehiy/pango/logman"

	"isrvd/internal/helper"
)

// FileHandler 文件处理器
type FileHandler struct{}

// NewFileHandler 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// List 文件列表
func (h *FileHandler) List(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("List files failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	fileList, err := fileList(path, req.Path)
	if err != nil {
		logman.Error("List files failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusNotFound, "Directory not found")
		return
	}

	helper.RespondSuccess(c, "Files listed successfully", gin.H{
		"path":  req.Path,
		"files": fileList,
	})
}

// Delete 删除文件
func (h *FileHandler) Delete(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Delete file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	if err := os.RemoveAll(path); err != nil {
		logman.Error("Delete file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}

	helper.RespondSuccess(c, "File deleted successfully", nil)
}

// Mkdir 创建目录
func (h *FileHandler) Mkdir(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create directory failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	if err := os.Mkdir(path, 0755); err != nil {
		logman.Error("Create directory failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create directory")
		return
	}

	helper.RespondSuccess(c, "Directory created successfully", nil)
}

// Create 新建文件
func (h *FileHandler) Create(c *gin.Context) {
	var req FileContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	if err := filer.Write(path, []byte(req.Content)); err != nil {
		logman.Error("Create file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}

	helper.RespondSuccess(c, "File created successfully", nil)
}

// Read 读取文件内容
func (h *FileHandler) Read(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Read file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	content, err := os.ReadFile(path)
	if err != nil {
		logman.Error("Read file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}

	helper.RespondSuccess(c, "File content retrieved", gin.H{
		"path":    req.Path,
		"content": string(content),
	})
}

// Modify 写入文件内容
func (h *FileHandler) Modify(c *gin.Context) {
	var req FileContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Modify file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		logman.Error("Modify file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot save file")
		return
	}

	helper.RespondSuccess(c, "File saved successfully", nil)
}

// Rename 重命名
func (h *FileHandler) Rename(c *gin.Context) {
	var req FileRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Rename file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	target := getAbsolutePath(c, filepath.Join(filepath.Dir(req.Path), req.Target))
	if err := os.Rename(path, target); err != nil {
		logman.Error("Rename file failed", "path", path, "target", target, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}

	helper.RespondSuccess(c, "File renamed successfully", nil)
}

// Chmod 修改权限
func (h *FileHandler) Chmod(c *gin.Context) {
	var req FileChmodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Chmod file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		logman.Error("Chmod file failed", "mode", req.Mode, "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid mode")
		return
	}

	path := getAbsolutePath(c, req.Path)
	if err = os.Chmod(path, os.FileMode(mode)); err != nil {
		logman.Error("Chmod file failed", "path", path, "mode", mode, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot change permissions")
		return
	}

	helper.RespondSuccess(c, "Permissions changed successfully", nil)
}

// Upload 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		logman.Error("Upload file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	path := c.PostForm("path")
	if path == "" {
		path = getAbsolutePath(c, header.Filename)
	} else {
		path = getAbsolutePath(c, filepath.Join(path, header.Filename))
	}

	f, err := os.Create(path)
	if err != nil {
		logman.Error("Create upload file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	defer f.Close()

	if _, err = io.Copy(f, file); err != nil {
		logman.Error("Write upload file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "Cannot write file")
		return
	}

	helper.RespondSuccess(c, "File uploaded successfully", nil)
}

// Download 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Download file failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "Invalid JSON")
		return
	}

	path := getAbsolutePath(c, req.Path)
	f, err := os.Open(path)
	if err != nil {
		logman.Error("Download file failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(req.Path))
	io.Copy(c.Writer, f)
}
