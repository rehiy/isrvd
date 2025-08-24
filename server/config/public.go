package config

import (
	"embed"
)

var Addr = ":8080"                // 监听地址
var BaseDir = "."                 // 基础目录
var UserMap = map[string]string{} // 用户名:明文密码

var PublicFS embed.FS // 静态文件
