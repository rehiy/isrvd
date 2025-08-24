package models

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 登录响应结构
type LoginResponse struct {
	Token string `json:"token"`
	User  string `json:"user"`
}
