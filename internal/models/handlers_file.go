package models

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Path string `json:"path"`
}

// DownloadRequest 下载文件请求
type DownloadRequest struct {
	Path string `json:"path"`
}

// DeleteRequest 删除文件请求
type DeleteRequest struct {
	Path string `json:"path"`
}

// ReadFileHandlerRequest 读取文件处理器请求
type ReadFileHandlerRequest struct {
	Path string `json:"path"`
}

// WriteFileHandlerRequest 写入文件处理器请求
type WriteFileHandlerRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ChmodHandlerRequest 修改权限处理器请求
type ChmodHandlerRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
}
