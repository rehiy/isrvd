package models

import "time"

// FileInfo 文件信息结构
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"isDir"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"modTime"`
	Path    string    `json:"path"`
}

// MkdirRequest 创建目录请求
type MkdirRequest struct {
	Path string `json:"path" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// NewFileRequest 新建文件请求
type NewFileRequest struct {
	Path    string `json:"path" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Content string `json:"content"`
}

// RenameRequest 重命名请求
type RenameRequest struct {
	Path    string `json:"path" binding:"required"`
	NewPath string `json:"newPath" binding:"required"`
}

// ChmodRequest 修改权限请求
type ChmodRequest struct {
	Mode string `json:"mode" binding:"required"`
}
