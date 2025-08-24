package model

import "time"

// 压缩请求
type ZipRequest struct {
	Path    string `json:"path" binding:"required"`
	ZipName string `json:"zipName" binding:"required"`
}

// 解压请求
type UnzipRequest struct {
	Path    string `json:"path" binding:"required"`
	ZipName string `json:"zipName" binding:"required"`
}

// zip信息请求
type ZipInfoRequest struct {
	Path string `json:"path" binding:"required"`
}

// zip文件信息
type ZipFileInfo struct {
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressedSize"`
	ModTime        time.Time `json:"modTime"`
	IsDir          bool      `json:"isDir"`
	CRC32          uint32    `json:"crc32"`
}

// 获取zip文件信息请求
type GetZipInfoRequest struct {
	Path string `json:"path"`
}

// 判断文件是否为zip文件请求
type IsZipFileRequest struct {
	Path string `json:"path"`
}
