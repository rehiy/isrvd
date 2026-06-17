package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/websocket"

	svcWebSSH "isrvd/internal/service/webssh"
)

// defineWebSSHRoutes 定义 WebSSH 模块路由
func (app *App) defineWebSSHRoutes() []Route {
	return []Route{
		// 认证凭据管理
		{Method: "GET", Path: "/ssh/credentials", Handler: app.websshCredentialList, Module: "ssh", Label: "查询 SSH 凭据列表"},
		{Method: "GET", Path: "/ssh/credential/:id", Handler: app.websshCredentialInspect, Module: "ssh", Label: "获取 SSH 凭据详情"},
		{Method: "POST", Path: "/ssh/credential", Handler: app.websshCredentialCreate, Module: "ssh", Label: "添加 SSH 凭据"},
		{Method: "PUT", Path: "/ssh/credential/:id", Handler: app.websshCredentialUpdate, Module: "ssh", Label: "更新 SSH 凭据"},
		{Method: "DELETE", Path: "/ssh/credential/:id", Handler: app.websshCredentialDelete, Module: "ssh", Label: "删除 SSH 凭据"},
		// 主机管理
		{Method: "GET", Path: "/ssh/hosts", Handler: app.websshHostList, Module: "ssh", Label: "查询 SSH 主机列表"},
		{Method: "GET", Path: "/ssh/host/:id", Handler: app.websshHostInspect, Module: "ssh", Label: "获取 SSH 主机详情"},
		{Method: "POST", Path: "/ssh/host", Handler: app.websshHostCreate, Module: "ssh", Label: "添加 SSH 主机"},
		{Method: "PUT", Path: "/ssh/host/:id", Handler: app.websshHostUpdate, Module: "ssh", Label: "更新 SSH 主机"},
		{Method: "DELETE", Path: "/ssh/host/:id", Handler: app.websshHostDelete, Module: "ssh", Label: "删除 SSH 主机"},
		// SSH 终端
		{Method: "GET", Path: "/ssh/to/:id", Handler: app.websshTerminal, Module: "ssh", Label: "打开 SSH 终端", QueryToken: true},
		// SFTP 文件管理
		{Method: "GET", Path: "/ssh/sftp/:id/ls", Handler: app.websshSFTPList, Module: "ssh", Label: "SFTP 列出目录"},
		{Method: "GET", Path: "/ssh/sftp/:id/download", Handler: app.websshSFTPDownload, Module: "ssh", Label: "SFTP 下载文件", QueryToken: true},
		{Method: "POST", Path: "/ssh/sftp/:id/upload", Handler: app.websshSFTPUpload, Module: "ssh", Label: "SFTP 上传文件"},
		{Method: "DELETE", Path: "/ssh/sftp/:id/rm", Handler: app.websshSFTPRemove, Module: "ssh", Label: "SFTP 删除文件"},
		{Method: "POST", Path: "/ssh/sftp/:id/mkdir", Handler: app.websshSFTPMkdir, Module: "ssh", Label: "SFTP 创建目录"},
		{Method: "POST", Path: "/ssh/sftp/:id/rename", Handler: app.websshSFTPRename, Module: "ssh", Label: "SFTP 重命名"},
		{Method: "POST", Path: "/ssh/sftp/:id/chmod", Handler: app.websshSFTPChmod, Module: "ssh", Label: "SFTP 修改权限"},
		{Method: "POST", Path: "/ssh/sftp/:id/chown", Handler: app.websshSFTPChown, Module: "ssh", Label: "SFTP 修改所有者"},
		{Method: "GET", Path: "/ssh/sftp/:id/read", Handler: app.websshSFTPRead, Module: "ssh", Label: "SFTP 读取文件"},
		{Method: "POST", Path: "/ssh/sftp/:id/write", Handler: app.websshSFTPWrite, Module: "ssh", Label: "SFTP 写入文件"},
		{Method: "GET", Path: "/ssh/sftp/:id/dir-size", Handler: app.websshSFTPDirSize, Module: "ssh", Label: "SFTP 计算目录大小"},
	}
}

// ─── Credential 凭据管理 ───

func (app *App) websshCredentialList(c *gin.Context) {
	respondSuccess(c, "", app.websshSvc.CredentialList())
}

func (app *App) websshCredentialInspect(c *gin.Context) {
	id := c.Param("id")
	cred := app.websshSvc.CredentialInspect(id)
	if cred == nil {
		respondError(c, http.StatusNotFound, "凭据不存在")
		return
	}
	respondSuccess(c, "", cred)
}

func (app *App) websshCredentialCreate(c *gin.Context) {
	var req svcWebSSH.CredentialUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	cred, err := app.websshSvc.CredentialCreate(&req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 凭据添加成功", cred)
}

func (app *App) websshCredentialUpdate(c *gin.Context) {
	id := c.Param("id")
	var req svcWebSSH.CredentialUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	cred, err := app.websshSvc.CredentialUpdate(id, &req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 凭据更新成功", cred)
}

func (app *App) websshCredentialDelete(c *gin.Context) {
	id := c.Param("id")
	if err := app.websshSvc.CredentialDelete(id); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 凭据删除成功", nil)
}

// ─── Host 主机管理 ───

// svcWebSSHHostUpsertRequest 是 service/webssh 中 HostUpsertRequest 的本地别名
type svcWebSSHHostUpsertRequest = svcWebSSH.HostUpsertRequest

func (app *App) websshHostList(c *gin.Context) {
	respondSuccess(c, "", app.websshSvc.HostList())
}

func (app *App) websshHostInspect(c *gin.Context) {
	id := c.Param("id")
	host := app.websshSvc.HostInspect(id)
	if host == nil {
		respondError(c, http.StatusNotFound, "主机不存在")
		return
	}
	respondSuccess(c, "", host)
}

func (app *App) websshHostCreate(c *gin.Context) {
	var req svcWebSSHHostUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	host, err := app.websshSvc.HostCreate(&req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机添加成功", host)
}

func (app *App) websshHostUpdate(c *gin.Context) {
	id := c.Param("id")
	var req svcWebSSHHostUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	host, err := app.websshSvc.HostUpdate(id, &req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机更新成功", host)
}

func (app *App) websshHostDelete(c *gin.Context) {
	id := c.Param("id")
	if err := app.websshSvc.HostDelete(id); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机删除成功", nil)
}

func (app *App) websshTerminal(c *gin.Context) {
	id := c.Param("id")
	app.wsConfig.Handler(func(conn *websocket.ServerConn) {
		app.websshSvc.RunTerminal(conn, id)
	})(c)
}

func (app *App) websshSFTPList(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.Query("path")
	result, err := app.websshSvc.SFTPList(id, dirPath)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

func (app *App) websshSFTPDownload(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}

	// 直接下载到 ResponseWriter
	err := app.websshSvc.SFTPDownload(id, filePath, c.Writer)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
}

func (app *App) websshSFTPUpload(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.Query("path")
	if dirPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		respondError(c, http.StatusBadRequest, "解析上传表单失败: "+err.Error())
		return
	}

	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		respondError(c, http.StatusBadRequest, "未找到上传文件")
		return
	}

	// relativePaths 与 files 数组一一对应（目录上传时前端传入 webkitRelativePath）
	relativePaths := c.Request.MultipartForm.Value["relativePath"]

	for i, header := range files {
		relativePath := header.Filename
		if i < len(relativePaths) && relativePaths[i] != "" {
			relativePath = relativePaths[i]
		}

		f, err := header.Open()
		if err != nil {
			respondError(c, http.StatusBadRequest, "打开文件失败: "+err.Error())
			return
		}
		uploadErr := app.websshSvc.SFTPUpload(id, dirPath, f, relativePath)
		f.Close()
		if uploadErr != nil {
			respondError(c, http.StatusBadRequest, uploadErr.Error())
			return
		}
	}

	respondSuccess(c, "上传成功", nil)
}

func (app *App) websshSFTPRemove(c *gin.Context) {
	id := c.Param("id")
	targetPath := c.Query("path")
	if targetPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	recursive := c.Query("recursive") == "true"
	if err := app.websshSvc.SFTPRemove(id, targetPath, recursive); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "删除成功", nil)
}

func (app *App) websshSFTPMkdir(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path string `json:"path" binding:"required"` // 要创建的目录路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.websshSvc.SFTPMkdir(id, req.Path); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "创建成功", nil)
}

func (app *App) websshSFTPRename(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		OldPath string `json:"oldPath" binding:"required"` // 原路径
		NewPath string `json:"newPath" binding:"required"` // 新路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.websshSvc.SFTPRename(id, req.OldPath, req.NewPath); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "重命名成功", nil)
}

// websshSFTPChmod 修改文件或目录权限
func (app *App) websshSFTPChmod(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path string `json:"path" binding:"required"` // 目标文件/目录路径
		Mode string `json:"mode" binding:"required"` // 权限模式，如 "0755", "0644"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 解析权限模式
	mode, err := parseFileMode(req.Mode)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的权限格式: "+err.Error())
		return
	}

	if err := app.websshSvc.SFTPChmod(id, req.Path, mode); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "权限修改成功", nil)
}

// websshSFTPChown 修改文件或目录所有者和组
func (app *App) websshSFTPChown(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path string `json:"path" binding:"required"` // 目标文件/目录路径
		UID  int    `json:"uid" binding:"required"`  // 新所有者用户 ID
		GID  int    `json:"gid" binding:"required"`  // 新所属组 ID
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := app.websshSvc.SFTPChown(id, req.Path, req.UID, req.GID); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "所有者修改成功", nil)
}

// parseFileMode 解析权限字符串（如 "0755", "644"）为 os.FileMode
func parseFileMode(modeStr string) (os.FileMode, error) {
	// 如果是 3-4 位的八进制数字字符串
	if len(modeStr) == 3 || len(modeStr) == 4 {
		var mode uint32
		if _, err := fmt.Sscanf(modeStr, "%o", &mode); err == nil {
			return os.FileMode(mode), nil
		}
	}
	return 0, fmt.Errorf("无法解析权限字符串: %s", modeStr)
}

// websshSFTPRead 读取文件内容
func (app *App) websshSFTPRead(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}

	content, err := app.websshSvc.SFTPRead(id, filePath)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, "", gin.H{
		"content": content,
	})
}

// websshSFTPWrite 写入文件内容
func (app *App) websshSFTPWrite(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path    string `json:"path" binding:"required"`    // 目标文件路径
		Content string `json:"content" binding:"required"` // 文件文本内容
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := app.websshSvc.SFTPWrite(id, req.Path, req.Content); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, "文件保存成功", nil)
}

// websshSFTPDirSize 计算远程目录大小
func (app *App) websshSFTPDirSize(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.Query("path")
	if dirPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}

	size, err := app.websshSvc.SFTPDirSize(id, dirPath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "无法计算目录大小: "+err.Error())
		return
	}

	respondSuccess(c, "计算目录大小成功", gin.H{"path": dirPath, "size": size})
}
