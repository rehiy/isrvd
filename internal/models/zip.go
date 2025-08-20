package models

import "time"

// ZipRequest 压缩请求
type ZipRequest struct {
	Path    string `json:"path" binding:"required"`
	ZipName string `json:"zipName" binding:"required"`
}

// UnzipRequest 解压请求
type UnzipRequest struct {
	Path    string `json:"path" binding:"required"`
	ZipName string `json:"zipName" binding:"required"`
}

// ZipInfoRequest zip信息请求
type ZipInfoRequest struct {
	Path string `json:"path" binding:"required"`
}

// ZipFileInfo zip文件信息
type ZipFileInfo struct {
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressedSize"`
	ModTime        time.Time `json:"modTime"`
	IsDir          bool      `json:"isDir"`
	CRC32          uint32    `json:"crc32"`
}
