package model

// 压缩请求
type ZipRequest struct {
	Path string `json:"path" binding:"required"`
}

// 解压请求
type UnzipRequest struct {
	Path string `json:"path" binding:"required"`
}
