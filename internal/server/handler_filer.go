package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/filer"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/helper"
	"isrvd/pkgs/archive"
)

// ─── 文件路径辅助 ───

func filerGetAbsPath(c *gin.Context, path string) string {
	home := filepath.Clean(filepath.Join(config.RootDirectory, "share"))
	if name := c.GetString("username"); name != "" {
		if member, ok := config.Members[name]; ok {
			home = filepath.Clean(member.HomeDirectory)
		}
	}
	abs := filepath.Clean(filepath.Join(home, path))
	rel, err := filepath.Rel(home, abs)
	if err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return abs
	}
	return home
}

type filerFileInfo struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModeO   string    `json:"modeO"`
	ModTime time.Time `json:"modTime"`
}

func filerFileList(path, rely string) ([]*filerFileInfo, error) {
	list, err := filer.List(path)
	if err != nil {
		return nil, err
	}
	var result []*filerFileInfo
	for _, f := range list {
		p := filepath.ToSlash(filepath.Join(rely, f.Name))
		result = append(result, &filerFileInfo{
			Path:    p,
			Name:    f.Name,
			Size:    f.Size,
			IsDir:   f.IsDir,
			Mode:    f.Mode.String(),
			ModeO:   strconv.FormatInt(int64(f.Mode), 8),
			ModTime: time.Unix(f.ModTime, 0),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir && !result[j].IsDir {
			return true
		}
		if !result[i].IsDir && result[j].IsDir {
			return false
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// ─── 请求结构 ───

type filerPathReq struct {
	Path string `json:"path" binding:"required"`
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

// ─── Handler 方法 ───

func (app *App) filerList(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	files, err := filerFileList(path, req.Path)
	if err != nil {
		logman.Error("List files failed", "path", path, "error", err)
		helper.RespondError(c, http.StatusNotFound, "Directory not found")
		return
	}
	helper.RespondSuccess(c, "Files listed successfully", gin.H{"path": req.Path, "files": files})
}

func (app *App) filerDelete(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	if err := os.RemoveAll(path); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot delete file")
		return
	}
	helper.RespondSuccess(c, "File deleted successfully", nil)
}

func (app *App) filerMkdir(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	if err := os.Mkdir(path, 0755); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create directory")
		return
	}
	helper.RespondSuccess(c, "Directory created successfully", nil)
}

func (app *App) filerCreate(c *gin.Context) {
	var req filerContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	if err := filer.Write(path, []byte(req.Content)); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	helper.RespondSuccess(c, "File created successfully", nil)
}

func (app *App) filerRead(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	content, err := os.ReadFile(path)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	helper.RespondSuccess(c, "File content retrieved", gin.H{"path": req.Path, "content": string(content)})
}

func (app *App) filerModify(c *gin.Context) {
	var req filerContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot save file")
		return
	}
	helper.RespondSuccess(c, "File saved successfully", nil)
}

func (app *App) filerRename(c *gin.Context) {
	var req filerRenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	target := filerGetAbsPath(c, filepath.Join(filepath.Dir(req.Path), req.Target))
	if err := os.Rename(path, target); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot rename file")
		return
	}
	helper.RespondSuccess(c, "File renamed successfully", nil)
}

func (app *App) filerChmod(c *gin.Context) {
	var req filerChmodReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "Invalid mode")
		return
	}
	path := filerGetAbsPath(c, req.Path)
	if err = os.Chmod(path, os.FileMode(mode)); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot change permissions")
		return
	}
	helper.RespondSuccess(c, "Permissions changed successfully", nil)
}

func (app *App) filerUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	path := c.PostForm("path")
	if path == "" {
		path = filerGetAbsPath(c, header.Filename)
	} else {
		path = filerGetAbsPath(c, filepath.Join(path, header.Filename))
	}

	f, err := os.Create(path)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot create file")
		return
	}
	defer f.Close()

	if _, err = io.Copy(f, file); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "Cannot write file")
		return
	}
	helper.RespondSuccess(c, "File uploaded successfully", nil)
}

func (app *App) filerDownload(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	path := filerGetAbsPath(c, req.Path)
	f, err := os.Open(path)
	if err != nil {
		helper.RespondError(c, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(req.Path))
	io.Copy(c.Writer, f)
}

func (app *App) filerZip(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	abs := filerGetAbsPath(c, req.Path)
	if err := archive.NewZipper().Zip(abs); err != nil {
		logman.Error("Create zip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法创建压缩文件")
		return
	}
	helper.RespondSuccess(c, "Archive created successfully", nil)
}

func (app *App) filerUnzip(c *gin.Context) {
	var req filerPathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	abs := filerGetAbsPath(c, req.Path)
	if err := archive.NewZipper().Unzip(abs); err != nil {
		logman.Error("Unzip failed", "path", abs, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "无法解压文件")
		return
	}
	helper.RespondSuccess(c, "Archive extracted successfully", nil)
}
