package config

import (
	"embed"
)

// 静态文件
var PublicFS embed.FS

// 监听地址
var ListenAddr = ":8080"

// 基础目录
var BaseDirectory = "."

// 用户名:明文密码
var Administrators = map[string]string{
	"admin": "admin",
}
