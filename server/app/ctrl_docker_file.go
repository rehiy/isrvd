package app

import (
	"errors"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"isrvd/pkgs/iobuf"
	"isrvd/server/config"
)

func (app *App) dockerContainerFileLs(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.DefaultQuery("path", "/")
	result, err := app.dockerSvc.ContainerFileList(c.Request.Context(), id, dirPath)
	respondResult(c, result, err)
}

func (app *App) dockerContainerFileDownload(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Content-Type", "application/octet-stream")
	if err := app.dockerSvc.ContainerFileDownload(c.Request.Context(), id, filePath, c.Writer); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
}

func (app *App) dockerContainerFileUpload(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.Query("path")
	if dirPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Current().Server.MaxUploadSize)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		respondError(c, http.StatusBadRequest, "解析上传表单失败: "+err.Error())
		return
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		respondError(c, http.StatusBadRequest, "未找到上传文件")
		return
	}
	relativePaths := c.Request.MultipartForm.Value["relativePath"]
	for i, header := range files {
		fileName := header.Filename
		if i < len(relativePaths) && relativePaths[i] != "" {
			fileName = relativePaths[i]
		}
		f, err := header.Open()
		if err != nil {
			respondError(c, http.StatusBadRequest, "打开文件失败: "+err.Error())
			return
		}
		uploadErr := app.dockerSvc.ContainerFileUpload(c.Request.Context(), id, dirPath, fileName, f)
		f.Close()
		if uploadErr != nil {
			respondError(c, http.StatusInternalServerError, uploadErr.Error())
			return
		}
	}
	respondSuccess(c, "上传成功", nil)
}

func (app *App) dockerContainerFileRemove(c *gin.Context) {
	id := c.Param("id")
	targetPath := c.Query("path")
	if targetPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	recursive := c.Query("recursive") == "true"
	err := app.dockerSvc.ContainerFileRemove(c.Request.Context(), id, targetPath, recursive)
	respondResultMsg(c, "删除成功", nil, err)
}

func (app *App) dockerContainerFileMkdir(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path string `json:"path" binding:"required"` // 要创建的目录路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ContainerFileMkdir(c.Request.Context(), id, req.Path)
	respondResultMsg(c, "创建成功", nil, err)
}

func (app *App) dockerContainerFileRename(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		OldPath string `json:"oldPath" binding:"required"` // 原路径
		NewPath string `json:"newPath" binding:"required"` // 新路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ContainerFileRename(c.Request.Context(), id, req.OldPath, req.NewPath)
	respondResultMsg(c, "重命名成功", nil, err)
}

func (app *App) dockerContainerFileRead(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	content, err := app.dockerSvc.ContainerFileRead(c.Request.Context(), id, filePath)
	if errors.Is(err, iobuf.ErrTooLarge) {
		respondError(c, http.StatusRequestEntityTooLarge, "文件超过在线编辑上限，请使用下载功能")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", gin.H{"content": content})
}

func (app *App) dockerContainerFileWrite(c *gin.Context) {
	id := c.Param("id")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxEditableJSONBytes)
	var req struct {
		Path    string `json:"path" binding:"required"` // 目标文件路径
		Content string `json:"content"`                 // 文件文本内容
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	if int64(len(req.Content)) > iobuf.MaxEditableFileBytes {
		respondError(c, http.StatusRequestEntityTooLarge, "文件超过在线编辑上限，请使用上传功能")
		return
	}
	err := app.dockerSvc.ContainerFileWrite(c.Request.Context(), id, req.Path, req.Content)
	respondResultMsg(c, "文件保存成功", nil, err)
}

func (app *App) dockerContainerFileChmod(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path string `json:"path" binding:"required"` // 目标文件/目录路径
		Mode string `json:"mode" binding:"required"` // 权限模式（如 "0644"）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.dockerSvc.ContainerFileChmod(c.Request.Context(), id, req.Path, req.Mode)
	respondResultMsg(c, "权限修改成功", nil, err)
}
