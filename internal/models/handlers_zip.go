package models

// GetZipInfoRequest 获取zip文件信息请求
type GetZipInfoRequest struct {
	Path string `json:"path"`
}

// IsZipFileRequest 判断文件是否为zip文件请求
type IsZipFileRequest struct {
	Path string `json:"path"`
}
