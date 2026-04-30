// Package helper 提供密码加密与验证工具函数
package helper

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 对密码进行哈希
// 空密码时直接返回空字符串，不进行加密
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("密码加密失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 验证密码是否匹配存储的哈希值
func VerifyPassword(password, hashedPassword string) bool {
	if hashedPassword == "" || password == "" {
		return hashedPassword == password
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// HashedBcrypt 检查密码是否是 bcrypt 格式
func HashedBcrypt(password string) bool {
	return len(password) >= 4 && password[0] == '$' && password[1] == '2' && password[3] == '$'
}
