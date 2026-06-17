package server

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/websocket"

	svcDocker "isrvd/internal/service/docker"
)

// defineDockerRoutes 定义 Docker 模块路由
func (app *App) defineDockerRoutes() []Route {
	return []Route{
		// Docker 服务
		{Method: "GET", Path: "/docker/info", Handler: app.dockerInfo, Module: "docker", Label: "获取 Docker 服务信息"},
		// 容器管理
		{Method: "GET", Path: "/docker/containers", Handler: app.dockerContainerList, Module: "docker", Label: "查询容器列表"},
		{Method: "GET", Path: "/docker/container/:id", Handler: app.dockerContainerInspect, Module: "docker", Label: "获取容器详情"},
		{Method: "POST", Path: "/docker/container", Handler: app.dockerContainerCreate, Module: "docker", Label: "创建容器"},
		{Method: "GET", Path: "/docker/container/:id/stats", Handler: app.dockerContainerStats, Module: "docker", Label: "获取容器资源统计"},
		{Method: "POST", Path: "/docker/container/:id/action", Handler: app.dockerContainerAction, Module: "docker", Label: "执行容器操作"},
		{Method: "GET", Path: "/docker/container/:id/logs", Handler: app.dockerContainerLogs, Module: "docker", Label: "获取容器日志"},
		{Method: "GET", Path: "/docker/container/:id/logs/stream", Handler: app.dockerContainerLogsStream, Module: "docker", Label: "实时查看容器日志", QueryToken: true},
		{Method: "GET", Path: "/docker/container/:id/exec", Handler: app.dockerContainerExec, Module: "docker", Label: "打开容器终端"},
		// 容器文件管理
		{Method: "GET", Path: "/docker/container/:id/file/ls", Handler: app.dockerContainerFileLs, Module: "docker", Label: "容器文件列目录"},
		{Method: "GET", Path: "/docker/container/:id/file/download", Handler: app.dockerContainerFileDownload, Module: "docker", Label: "容器文件下载", QueryToken: true},
		{Method: "POST", Path: "/docker/container/:id/file/upload", Handler: app.dockerContainerFileUpload, Module: "docker", Label: "容器文件上传"},
		{Method: "DELETE", Path: "/docker/container/:id/file/rm", Handler: app.dockerContainerFileRemove, Module: "docker", Label: "容器文件删除"},
		{Method: "POST", Path: "/docker/container/:id/file/mkdir", Handler: app.dockerContainerFileMkdir, Module: "docker", Label: "容器文件创建目录"},
		{Method: "POST", Path: "/docker/container/:id/file/rename", Handler: app.dockerContainerFileRename, Module: "docker", Label: "容器文件重命名"},
		{Method: "GET", Path: "/docker/container/:id/file/read", Handler: app.dockerContainerFileRead, Module: "docker", Label: "容器文件读取"},
		{Method: "POST", Path: "/docker/container/:id/file/write", Handler: app.dockerContainerFileWrite, Module: "docker", Label: "容器文件写入"},
		{Method: "POST", Path: "/docker/container/:id/file/chmod", Handler: app.dockerContainerFileChmod, Module: "docker", Label: "容器文件修改权限"},
		// 镜像管理
		{Method: "GET", Path: "/docker/images", Handler: app.dockerImageList, Module: "docker", Label: "查询镜像列表"},
		{Method: "GET", Path: "/docker/images/search", Handler: app.dockerImageSearch, Module: "docker", Label: "搜索镜像"},
		{Method: "POST", Path: "/docker/image/:id/action", Handler: app.dockerImageAction, Module: "docker", Label: "执行镜像操作"},
		{Method: "POST", Path: "/docker/image/:id/tag", Handler: app.dockerImageTag, Module: "docker", Label: "添加镜像标签"},
		{Method: "GET", Path: "/docker/image/:id", Handler: app.dockerImageInspect, Module: "docker", Label: "获取镜像详情"},
		{Method: "POST", Path: "/docker/image/build", Handler: app.dockerImageBuild, Module: "docker", Label: "构建镜像"},
		{Method: "POST", Path: "/docker/image/prune", Handler: app.dockerImagePrune, Module: "docker", Label: "清理镜像"},
		{Method: "POST", Path: "/docker/image/push", Handler: app.dockerImagePush, Module: "docker", Label: "推送镜像"},
		{Method: "POST", Path: "/docker/image/pull", Handler: app.dockerImagePull, Module: "docker", Label: "拉取镜像"},
		// 网络管理
		{Method: "GET", Path: "/docker/networks", Handler: app.dockerNetworkList, Module: "docker", Label: "查询网络列表"},
		{Method: "POST", Path: "/docker/network/:id/action", Handler: app.dockerNetworkAction, Module: "docker", Label: "执行网络操作"},
		{Method: "POST", Path: "/docker/network", Handler: app.dockerNetworkCreate, Module: "docker", Label: "创建网络"},
		{Method: "GET", Path: "/docker/network/:id", Handler: app.dockerNetworkInspect, Module: "docker", Label: "获取网络详情"},
		// 数据卷管理
		{Method: "GET", Path: "/docker/volumes", Handler: app.dockerVolumeList, Module: "docker", Label: "查询数据卷列表"},
		{Method: "POST", Path: "/docker/volume/:name/action", Handler: app.dockerVolumeAction, Module: "docker", Label: "执行数据卷操作"},
		{Method: "POST", Path: "/docker/volume", Handler: app.dockerVolumeCreate, Module: "docker", Label: "创建数据卷"},
		{Method: "GET", Path: "/docker/volume/:name", Handler: app.dockerVolumeInspect, Module: "docker", Label: "获取数据卷详情"},
		// 镜像仓库
		{Method: "GET", Path: "/docker/registries", Handler: app.dockerRegistryList, Module: "docker", Label: "查询镜像仓库列表"},
		{Method: "POST", Path: "/docker/registry", Handler: app.dockerRegistryCreate, Module: "docker", Label: "添加镜像仓库"},
		{Method: "PUT", Path: "/docker/registry", Handler: app.dockerRegistryUpdate, Module: "docker", Label: "更新镜像仓库"},
		{Method: "DELETE", Path: "/docker/registry", Handler: app.dockerRegistryDelete, Module: "docker", Label: "删除镜像仓库"},
	}
}

func (app *App) dockerInfo(c *gin.Context) {
	result, err := app.dockerSvc.Info(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 Docker 服务信息成功", result)
}

func (app *App) dockerContainerList(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"
	result, err := app.dockerSvc.ContainerList(c.Request.Context(), all)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取容器列表成功", result)
}

func (app *App) dockerContainerCreate(c *gin.Context) {
	var req svcDocker.ContainerSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ContainerCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "容器创建成功", result)
}

func (app *App) dockerContainerInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ContainerInspect(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取容器详情成功", result)
}

func (app *App) dockerContainerStats(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ContainerStats(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取容器资源统计成功", result)
}

func (app *App) dockerContainerAction(c *gin.Context) {
	req := svcDocker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.ContainerAction(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "容器操作成功", nil)
}

func (app *App) dockerContainerLogs(c *gin.Context) {
	req := svcDocker.ContainerLogsRequest{
		ID:   c.Param("id"),
		Tail: c.DefaultQuery("tail", "100"),
	}
	if req.ID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}
	result, err := app.dockerSvc.ContainerLogs(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取容器日志成功", result)
}

func (app *App) dockerContainerLogsStream(c *gin.Context) {
	req := svcDocker.ContainerLogsRequest{
		ID:   c.Param("id"),
		Tail: c.DefaultQuery("tail", "100"),
	}
	if req.ID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}
	w, err := httpd.NewEventWriter(c.Writer)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	app.dockerSvc.ContainerLogsStream(c.Request.Context(), w, req)
}

func (app *App) dockerContainerExec(c *gin.Context) {
	containerID := c.Param("id")
	shell := c.DefaultQuery("shell", "/bin/sh")
	if containerID == "" {
		respondError(c, http.StatusBadRequest, "缺少容器 ID")
		return
	}

	// 使用 Handler 模式处理 WebSocket
	app.wsConfig.Handler(func(conn *websocket.ServerConn) {
		app.dockerSvc.ContainerExec(c.Request.Context(), conn, containerID, shell)
	})(c)
}

// ─── 镜像 ───

func (app *App) dockerImageList(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"
	result, err := app.dockerSvc.ImageList(c.Request.Context(), all)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取镜像列表成功", result)
}

func (app *App) dockerImageAction(c *gin.Context) {
	req := svcDocker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.ImageAction(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "镜像操作成功", nil)
}

func (app *App) dockerImageTag(c *gin.Context) {
	req := svcDocker.ImageTagRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.ImageTag(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "添加镜像标签成功", nil)
}

func (app *App) dockerImageSearch(c *gin.Context) {
	name := c.Query("name")
	result, err := app.dockerSvc.ImageSearch(c.Request.Context(), name)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "搜索镜像成功", result)
}

func (app *App) dockerImageBuild(c *gin.Context) {
	var req svcDocker.ImageBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImageBuild(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "镜像构建成功", result)
}

func (app *App) dockerImagePrune(c *gin.Context) {
	var req svcDocker.ImagePruneRequest
	// 请求体可选；空 JSON 表示仅清理悬空层
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePrune(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "镜像清理成功", result)
}

func (app *App) dockerImageInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.ImageInspect(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取镜像详情成功", result)
}

// ─── 网络 ───

func (app *App) dockerNetworkList(c *gin.Context) {
	result, err := app.dockerSvc.NetworkList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取网络列表成功", result)
}

func (app *App) dockerNetworkAction(c *gin.Context) {
	req := svcDocker.ActionRequest{
		ID: c.Param("id"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.NetworkAction(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "网络操作成功", nil)
}

func (app *App) dockerNetworkCreate(c *gin.Context) {
	var req svcDocker.NetworkSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.NetworkCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "网络创建成功", result)
}

func (app *App) dockerNetworkInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.dockerSvc.NetworkInspect(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取网络详情成功", result)
}

// ─── 数据卷 ───

func (app *App) dockerVolumeList(c *gin.Context) {
	result, err := app.dockerSvc.VolumeList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取数据卷列表成功", result)
}

func (app *App) dockerVolumeAction(c *gin.Context) {
	req := svcDocker.ActionRequest{
		ID: c.Param("name"),
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.VolumeAction(c.Request.Context(), req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "数据卷操作成功", nil)
}

func (app *App) dockerVolumeCreate(c *gin.Context) {
	var req svcDocker.VolumeSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.VolumeCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "数据卷创建成功", result)
}

func (app *App) dockerVolumeInspect(c *gin.Context) {
	name := c.Param("name")
	result, err := app.dockerSvc.VolumeInspect(c.Request.Context(), name)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取数据卷详情成功", result)
}

// ─── 镜像仓库 ───

// svcDockerRegistryUpsertRequest 是 service/docker 中 RegistryUpsertRequest 的本地别名
type svcDockerRegistryUpsertRequest = svcDocker.RegistryUpsertRequest

func (app *App) dockerRegistryList(c *gin.Context) {
	respondSuccess(c, "获取镜像仓库列表成功", app.dockerSvc.RegistryList())
}

func (app *App) dockerRegistryCreate(c *gin.Context) {
	var req svcDockerRegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.RegistryCreate(req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库添加成功", nil)
}

func (app *App) dockerRegistryUpdate(c *gin.Context) {
	originalURL := c.Query("url")
	if originalURL == "" {
		respondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	var req svcDockerRegistryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.RegistryUpdate(originalURL, req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库更新成功", nil)
}

func (app *App) dockerRegistryDelete(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		respondError(c, http.StatusBadRequest, "缺少 url 参数")
		return
	}
	if err := app.dockerSvc.RegistryDelete(url); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "镜像仓库删除成功", nil)
}

func (app *App) dockerImagePush(c *gin.Context) {
	var req svcDocker.ImagePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePush(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "镜像推送成功", result)
}

func (app *App) dockerImagePull(c *gin.Context) {
	var req svcDocker.ImagePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.dockerSvc.ImagePull(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "镜像拉取成功", result)
}

// ─── 容器文件管理 ───

func (app *App) dockerContainerFileLs(c *gin.Context) {
	id := c.Param("id")
	dirPath := c.DefaultQuery("path", "/")
	result, err := app.dockerSvc.ContainerFileList(c.Request.Context(), id, dirPath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
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
	if err := app.dockerSvc.ContainerFileRemove(c.Request.Context(), id, targetPath, recursive); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "删除成功", nil)
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
	if err := app.dockerSvc.ContainerFileMkdir(c.Request.Context(), id, req.Path); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "创建成功", nil)
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
	if err := app.dockerSvc.ContainerFileRename(c.Request.Context(), id, req.OldPath, req.NewPath); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "重命名成功", nil)
}

func (app *App) dockerContainerFileRead(c *gin.Context) {
	id := c.Param("id")
	filePath := c.Query("path")
	if filePath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	content, err := app.dockerSvc.ContainerFileRead(c.Request.Context(), id, filePath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", gin.H{"content": content})
}

func (app *App) dockerContainerFileWrite(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Path    string `json:"path" binding:"required"` // 目标文件路径
		Content string `json:"content"`                 // 文件文本内容
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.dockerSvc.ContainerFileWrite(c.Request.Context(), id, req.Path, req.Content); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "文件保存成功", nil)
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
	if err := app.dockerSvc.ContainerFileChmod(c.Request.Context(), id, req.Path, req.Mode); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "权限修改成功", nil)
}
