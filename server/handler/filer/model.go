package filer

// FileRequest 文件路径请求
type FileRequest struct {
	Path string `json:"path" binding:"required"`
}

// FileContentRequest 新建文件请求
type FileContentRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// FileChmodRequest 修改权限请求
type FileChmodRequest struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
}

// FileRenameRequest 重命名请求
type FileRenameRequest struct {
	Path   string `json:"path" binding:"required"`
	Target string `json:"target" binding:"required"`
}
