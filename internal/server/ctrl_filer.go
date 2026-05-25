package server

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// defineFilerRoutes 定义 Filer 模块路由（文件管理）
func (app *App) defineFilerRoutes() []Route {
	return []Route{
		// 读取类操作不记录审计日志
		{Method: "GET", Path: "/filer/list", Handler: app.filerFileList, Module: "filer", Label: "查询目录文件列表"},
		{Method: "GET", Path: "/filer/read", Handler: app.filerFileRead, Module: "filer", Label: "读取文件内容"},
		{Method: "GET", Path: "/filer/download", Handler: app.filerFileDownload, Module: "filer", Label: "下载文件", QueryToken: true},
		// 写入与变更类操作按默认策略审计
		{Method: "POST", Path: "/filer/mkdir", Handler: app.filerFileMkdir, Module: "filer", Label: "创建目录"},
		{Method: "POST", Path: "/filer/create", Handler: app.filerFileCreate, Module: "filer", Label: "创建文件"},
		{Method: "POST", Path: "/filer/modify", Handler: app.filerFileModify, Module: "filer", Label: "保存文件内容"},
		{Method: "POST", Path: "/filer/rename", Handler: app.filerFileRename, Module: "filer", Label: "重命名文件或目录"},
		{Method: "POST", Path: "/filer/delete", Handler: app.filerFileDelete, Module: "filer", Label: "删除文件或目录"},
		{Method: "POST", Path: "/filer/chmod", Handler: app.filerFileChmod, Module: "filer", Label: "修改文件权限"},
		{Method: "POST", Path: "/filer/upload", Handler: app.filerFileUpload, Module: "filer", Label: "上传文件"},
		{Method: "POST", Path: "/filer/zip", Handler: app.filerFileZip, Module: "filer", Label: "压缩文件或目录"},
		{Method: "POST", Path: "/filer/unzip", Handler: app.filerFileUnzip, Module: "filer", Label: "解压文件"},
	}
}

// ─── 请求结构 ───

type filerPathReq struct {
	Path string `json:"path" form:"path" binding:"required"`
}

type filerContentReq struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type filerChmodReq struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
}

type filerRenameReq struct {
	Path   string `json:"path" binding:"required"`
	Target string `json:"target" binding:"required"`
}

var filerPreviewContentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".m4a":  "audio/mp4",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".m4v":  "video/x-m4v",
	".mkv":  "video/x-matroska",
	".pdf":  "application/pdf",
}

func (app *App) filerAbsPath(c *gin.Context, path string) (string, bool) {
	absPath, err := app.filerSvc.AbsPath(c.GetString("username"), path)
	if err != nil {
		respondError(c, http.StatusForbidden, err.Error())
		return "", false
	}
	return absPath, true
}

// ─── Handler 方法 ───

func (app *App) filerFileList(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	files, err := app.filerSvc.FileList(absPath, req.Path)
	if err != nil {
		logman.Error("List files failed", "path", absPath, "error", err)
		respondError(c, http.StatusNotFound, "Directory not found")
		return
	}
	respondSuccess(c, "Files listed successfully", gin.H{"path": req.Path, "files": files})
}

func (app *App) filerFileDelete(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileDelete(absPath); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}
	respondSuccess(c, "File deleted successfully", nil)
}

func (app *App) filerFileMkdir(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileMkdir(absPath); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot create directory")
		return
	}
	respondSuccess(c, "Directory created successfully", nil)
}

func (app *App) filerFileCreate(c *gin.Context) {
	var req filerContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileCreate(absPath, []byte(req.Content)); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	respondSuccess(c, "File created successfully", nil)
}

func (app *App) filerFileRead(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	content, err := app.filerSvc.FileRead(absPath)
	if err != nil {
		respondError(c, http.StatusNotFound, "File not found")
		return
	}
	respondSuccess(c, "File content retrieved", gin.H{"path": req.Path, "content": string(content)})
}

func (app *App) filerFileModify(c *gin.Context) {
	var req filerContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileWrite(absPath, []byte(req.Content)); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot save file")
		return
	}
	respondSuccess(c, "File saved successfully", nil)
}

func (app *App) filerFileRename(c *gin.Context) {
	var req filerRenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	targetPath, ok := app.filerAbsPath(c, filepath.Join(filepath.Dir(req.Path), req.Target))
	if !ok {
		return
	}

	if err := app.filerSvc.FileRename(absPath, targetPath); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}
	respondSuccess(c, "File renamed successfully", nil)
}

func (app *App) filerFileChmod(c *gin.Context) {
	var req filerChmodReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileChmod(absPath, req.Mode); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "Permissions changed successfully", nil)
}

func (app *App) filerFileUpload(c *gin.Context) {
	if c.Request.ContentLength > config.Server.MaxUploadSize {
		respondError(c, http.StatusBadRequest, "文件大小超过限制")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Server.MaxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	path := c.PostForm("path")
	if path == "" {
		path = header.Filename
	} else {
		path = filepath.Join(path, header.Filename)
	}

	absPath, ok := app.filerAbsPath(c, path)
	if !ok {
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot read uploaded file")
		return
	}
	if err := app.filerSvc.FileWrite(absPath, data); err != nil {
		respondError(c, http.StatusInternalServerError, "Cannot write file")
		return
	}
	respondSuccess(c, "File uploaded successfully", nil)
}

func (app *App) filerFileDownload(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	inline := c.Query("inline") == "1"
	contentType := ""
	if inline {
		var supported bool
		contentType, supported = filerPreviewContentTypes[strings.ToLower(filepath.Ext(req.Path))]
		if !supported {
			respondError(c, http.StatusUnsupportedMediaType, "Unsupported preview file type")
			return
		}
	}

	file, info, err := app.filerSvc.FileOpen(absPath)
	if err != nil {
		respondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer file.Close()

	filename := filepath.Base(req.Path)
	if inline {
		c.Header("Content-Type", contentType)
		c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
		c.Header("X-Frame-Options", "SAMEORIGIN")
	} else {
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	}
	http.ServeContent(c.Writer, c.Request, filename, info.ModTime(), file)
}

func (app *App) filerFileZip(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileZip(absPath); err != nil {
		logman.Error("Create zip failed", "path", absPath, "error", err)
		respondError(c, http.StatusInternalServerError, "无法创建压缩文件")
		return
	}
	respondSuccess(c, "Archive created successfully", nil)
}

func (app *App) filerFileUnzip(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileUnzip(absPath); err != nil {
		logman.Error("Unzip failed", "path", absPath, "error", err)
		respondError(c, http.StatusInternalServerError, "无法解压文件")
		return
	}
	respondSuccess(c, "Archive extracted successfully", nil)
}
