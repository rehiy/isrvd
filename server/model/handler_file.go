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

// 文件路径请求
type FileRequest struct {
	Path string `json:"path" binding:"required"`
}

// 新建文件请求
type FileContentRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// 修改权限请求
type FileChmodRequest struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
}

// 重命名请求
type FileRenameRequest struct {
	Path   string `json:"path" binding:"required"`
	Target string `json:"target" binding:"required"`
}
