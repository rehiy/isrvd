package model

import "time"

// 文件信息结构
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"modTime"`
	Path    string    `json:"path"`
}

// 创建目录请求
type MkdirRequest struct {
	Path string `json:"path" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// 新建文件请求
type NewFileRequest struct {
	Path    string `json:"path" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Content string `json:"content"`
}

// 重命名请求
type RenameRequest struct {
	Path    string `json:"path" binding:"required"`
	NewPath string `json:"newPath" binding:"required"`
}

// 修改权限请求
type ChmodRequest struct {
	Mode string `json:"mode" binding:"required"`
}

// 文件列表请求
type ListFilesRequest struct {
	Path string `json:"path"`
}

// 下载文件请求
type DownloadRequest struct {
	Path string `json:"path"`
}

// 删除文件请求
type DeleteRequest struct {
	Path string `json:"path"`
}

// 读取文件处理器请求
type ReadFileHandlerRequest struct {
	Path string `json:"path"`
}

// 写入文件处理器请求
type WriteFileHandlerRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// 修改权限处理器请求
type ChmodHandlerRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
}
