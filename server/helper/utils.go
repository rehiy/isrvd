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

// 获取绝对路径
func GetAbsolutePath(path string) string {
	cfg := config.GetGlobal()
	return filepath.Join(cfg.BaseDir, filepath.Clean(path))
}

// 验证路径安全性，防止目录遍历攻击
func ValidatePath(path string) bool {
	cleanPath := filepath.Clean(path)
	return !strings.Contains(cleanPath, "..")
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
