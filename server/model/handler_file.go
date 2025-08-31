package model

import "time"

// 文件信息结构
type FileInfo struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModeO   string    `json:"modeO"`
	ModTime time.Time `json:"modTime"`
}

// 创建目录请求
type MkdirRequest struct {
	Path string `json:"path"`
	Name string `json:"name" binding:"required"`
}

// 新建文件请求
type CreateRequest struct {
	Path    string `json:"path"`
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// 重命名请求
type RenameRequest struct {
	Path   string `json:"path" binding:"required"`
	Target string `json:"target" binding:"required"`
}

// 修改权限请求
type ChmodRequest struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
}

// 文件列表请求
type ListRequest struct {
	Path string `json:"path"`
}

// 下载文件请求
type DownloadRequest struct {
	Path string `json:"path" binding:"required"`
}

// 删除文件请求
type DeleteRequest struct {
	Path string `json:"path" binding:"required"`
}

// 读取文件请求
type ReadRequest struct {
	Path string `json:"path" binding:"required"`
}

// 写入文件请求
type ModifyRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
}
