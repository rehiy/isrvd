package helper

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"isrvd/server/config"
)

// 计算MD5哈希
func Md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// 验证路径安全性，防止目录遍历攻击
func ValidatePath(path string) bool {
	return !strings.Contains(filepath.Clean(path), "..")
}

// 获取用户的绝对路径
func GetAbsolutePath(c *gin.Context, path string) string {
	username := GetCurrentUser(c)
	var basePath string
	if member, exists := config.Members[username]; exists {
		basePath = member.HomeDirectory
	} else {
		basePath = filepath.Join(config.RootDirectory, "share")
	}

	// 规范化并拼接路径
	absPath := filepath.Join(basePath, filepath.Clean(path))

	// 验证最终路径是否在允许的基础目录下（防止目录遍历攻击）
	if !strings.HasPrefix(absPath, basePath) {
		return basePath
	}

	return absPath
}

// 从请求中获取令牌
func GetTokenFromRequest(c *gin.Context) string {
	// 优先从 Authorization 头获取
	token := c.GetHeader("Authorization")
	if token != "" {
		// 移除 "Bearer " 前缀（如果存在）
		token = strings.TrimPrefix(token, "Bearer ")
		return token
	}

	// 从查询参数获取
	return c.Query("token")
}

// 从请求中获取当前用户
func GetCurrentUser(c *gin.Context) string {
	token := GetTokenFromRequest(c)
	if token == "" {
		return ""
	}

	for username, member := range config.Members {
		if token == member.Token {
			return username
		}
	}
	return ""
}
