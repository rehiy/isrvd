package server

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// defineFilerRoutes 定义 Filer 模块路由（RESTful）
func (app *App) defineFilerRoutes() []Route {
	return []Route{
		// 读取类操作（GET）
		{Method: "GET", Path: "/filer/files", Handler: app.filerFileList, Module: "filer", Label: "查询目录文件列表"},
		{Method: "GET", Path: "/filer/file", Handler: app.filerFileRead, Module: "filer", Label: "读取文件内容"},
		{Method: "GET", Path: "/filer/download", Handler: app.filerFileDownload, Module: "filer", Label: "下载文件", QueryToken: true},
		{Method: "GET", Path: "/filer/dir-size", Handler: app.filerDirSize, Module: "filer", Label: "计算目录大小"},
		// 创建类操作（POST）
		{Method: "POST", Path: "/filer/dir", Handler: app.filerFileMkdir, Module: "filer", Label: "创建目录"},
		{Method: "POST", Path: "/filer/file", Handler: app.filerFileCreate, Module: "filer", Label: "创建文件"},
		{Method: "POST", Path: "/filer/upload", Handler: app.filerFileUpload, Module: "filer", Label: "上传文件"},
		// 修改类操作（PUT）
		{Method: "PUT", Path: "/filer/file", Handler: app.filerFileModify, Module: "filer", Label: "保存文件内容"},
		{Method: "PUT", Path: "/filer/chmod", Handler: app.filerFileChmod, Module: "filer", Label: "修改文件权限"},
		// 删除类操作（DELETE）
		{Method: "DELETE", Path: "/filer/file", Handler: app.filerFileDelete, Module: "filer", Label: "删除文件或目录"},
		// 动作型操作（POST）
		{Method: "POST", Path: "/filer/rename", Handler: app.filerFileRename, Module: "filer", Label: "重命名文件或目录"},
		{Method: "POST", Path: "/filer/zip", Handler: app.filerFileZip, Module: "filer", Label: "压缩文件或目录"},
		{Method: "POST", Path: "/filer/unzip", Handler: app.filerFileUnzip, Module: "filer", Label: "解压文件"},
	}
}

// ─── Handler 方法 ───

type filerPathQuery struct {
	Path string `form:"path" binding:"required"` // 文件或目录路径
}

func (app *App) filerFileList(c *gin.Context) {
	var req filerPathQuery
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
		respondError(c, http.StatusNotFound, "目录不存在")
		return
	}
	respondSuccess(c, "获取文件列表成功", gin.H{"path": req.Path, "files": files})
}

func (app *App) filerDirSize(c *gin.Context) {
	var req filerPathQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	size, err := app.filerSvc.DirSize(absPath)
	if err != nil {
		logman.Error("Calculate dir size failed", "path", absPath, "error", err)
		respondError(c, http.StatusInternalServerError, "无法计算目录大小")
		return
	}
	respondSuccess(c, "计算目录大小成功", gin.H{"path": req.Path, "size": size})
}

func (app *App) filerFileDelete(c *gin.Context) {
	var req filerPathQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileDelete(absPath); err != nil {
		respondError(c, http.StatusInternalServerError, "无法删除文件")
		return
	}
	respondSuccess(c, "文件删除成功", nil)
}

func (app *App) filerFileMkdir(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"` // 要创建的目录路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileMkdir(absPath); err != nil {
		respondError(c, http.StatusInternalServerError, "无法创建目录")
		return
	}
	respondSuccess(c, "目录创建成功", nil)
}

type filerContentBody struct {
	Path    string `json:"path" binding:"required"`    // 文件路径
	Content string `json:"content" binding:"required"` // 文件内容
}

func (app *App) filerFileCreate(c *gin.Context) {
	var req filerContentBody
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileCreate(absPath, []byte(req.Content)); err != nil {
		respondError(c, http.StatusInternalServerError, "无法创建文件")
		return
	}
	respondSuccess(c, "文件创建成功", nil)
}

func (app *App) filerFileRead(c *gin.Context) {
	var req filerPathQuery
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
		respondError(c, http.StatusNotFound, "文件未找到")
		return
	}
	respondSuccess(c, "获取文件内容成功", gin.H{"path": req.Path, "content": string(content)})
}

func (app *App) filerFileModify(c *gin.Context) {
	var req filerContentBody
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	if err := app.filerSvc.FileWrite(absPath, []byte(req.Content)); err != nil {
		respondError(c, http.StatusInternalServerError, "无法保存文件")
		return
	}
	respondSuccess(c, "文件保存成功", nil)
}

type filerRenameBody struct {
	Path   string `json:"path" binding:"required"`   // 原路径
	Target string `json:"target" binding:"required"` // 目标路径或名称
}

func (app *App) filerFileRename(c *gin.Context) {
	var req filerRenameBody
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	target := req.Target
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(req.Path), target)
	}
	targetPath, ok := app.filerAbsPath(c, target)
	if !ok {
		return
	}

	if err := app.filerSvc.FileRename(absPath, targetPath); err != nil {
		respondError(c, http.StatusInternalServerError, "无法重命名文件")
		return
	}
	respondSuccess(c, "文件重命名成功", nil)
}

type filerChmodBody struct {
	Path string `json:"path" binding:"required"` // 文件路径
	Mode string `json:"mode" binding:"required"` // 权限模式（如 "0755"）
}

func (app *App) filerFileChmod(c *gin.Context) {
	var req filerChmodBody
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
	respondSuccess(c, "权限修改成功", nil)
}

func (app *App) filerFileUpload(c *gin.Context) {
	if c.Request.ContentLength > config.Server.MaxUploadSize {
		respondError(c, http.StatusBadRequest, "文件大小超过限制")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Server.MaxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "未上传文件")
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
		respondError(c, http.StatusInternalServerError, "无法读取上传文件")
		return
	}
	if err := app.filerSvc.FileWrite(absPath, data); err != nil {
		respondError(c, http.StatusInternalServerError, "无法写入文件")
		return
	}
	respondSuccess(c, "文件上传成功", nil)
}

func (app *App) filerFileDownload(c *gin.Context) {
	var req struct {
		Path   string `form:"path" binding:"required"` // 要下载的文件路径
		Inline bool   `form:"inline"`                  // true 内联预览（Content-Disposition: inline），false 触发下载
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	inline := req.Inline
	contentType := ""
	if inline {
		contentType = app.filerSvc.PreviewContentType(filepath.Ext(req.Path))
		if contentType == "" {
			respondError(c, http.StatusUnsupportedMediaType, "不支持预览的文件类型")
			return
		}
	}

	file, info, err := app.filerSvc.FileOpen(absPath)
	if err != nil {
		respondError(c, http.StatusNotFound, "文件未找到")
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
	var req filerPathQuery
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
	respondSuccess(c, "压缩文件创建成功", nil)
}

type filerUnzipBody struct {
	Path      string `json:"path" binding:"required"` // 压缩文件路径
	TargetDir string `json:"targetDir"`               // 可选：指定解压目标目录名（仅允许标准目录名，无 / 等分隔符）
}

func (app *App) filerFileUnzip(c *gin.Context) {
	var req filerUnzipBody
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	absPath, ok := app.filerAbsPath(c, req.Path)
	if !ok {
		return
	}

	// 如果指定了目标目录名，验证其有效性并拼接到 zip 文件当前目录
	var targetPath string
	if req.TargetDir != "" {
		// 验证目录名：不能是绝对路径、不能含路径分隔符、不能是 . 或 ..
		if filepath.IsAbs(req.TargetDir) || filepath.Base(req.TargetDir) != req.TargetDir || req.TargetDir == "." || req.TargetDir == ".." {
			respondError(c, http.StatusBadRequest, "无效的目录名，不允许包含路径分隔符")
			return
		}
		// 拼接到 zip 文件的当前目录
		targetPath = filepath.Join(filepath.Dir(absPath), req.TargetDir)
	}

	if err := app.filerSvc.FileUnzip(absPath, targetPath); err != nil {
		logman.Error("Unzip failed", "path", absPath, "targetDir", req.TargetDir, "error", err)
		respondError(c, http.StatusInternalServerError, "无法解压文件")
		return
	}
	respondSuccess(c, "文件解压成功", nil)
}

// ─── 内部方法 ───

func (app *App) filerAbsPath(c *gin.Context, path string) (string, bool) {
	absPath, err := app.filerSvc.AbsPath(c.GetString("username"), path)
	if err != nil {
		respondError(c, http.StatusForbidden, err.Error())
		return "", false
	}
	return absPath, true
}
