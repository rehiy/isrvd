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
	username := c.GetString("username")

	var homePath string
	if member, exists := config.Members[username]; exists {
		homePath = member.HomeDirectory
	} else {
		homePath = filepath.Join(config.RootDirectory, "share")
	}

	// 规范化并拼接路径
	// 验证最终路径是否在允许的基础目录下（防止目录遍历攻击）
	absPath := filepath.Join(homePath, filepath.Clean(path))
	if !strings.HasPrefix(absPath, homePath) {
		return homePath
	}
	return absPath
}
