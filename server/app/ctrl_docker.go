package app

import (
	"github.com/gin-gonic/gin"
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
	respondResultMsg(c, "获取 Docker 服务信息成功", result, err)
}
