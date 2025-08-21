package models

// APIResponse 通用API响应结构
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
	Payload    any   `json:"payload,omitempty"`
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
