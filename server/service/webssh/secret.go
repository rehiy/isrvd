package webssh

import (
	"crypto/cipher"
	"errors"
	"strings"

	"isrvd/pkgs/secretbox"
	"isrvd/server/config"
)

// SSH 密码/私钥轻量落盘加密：密钥由 JWT 密钥派生，仅防止文件被直接查看时泄露明文。
// 无前缀的值视为历史明文，下次保存时自动加密。

func secretAEAD() (cipher.AEAD, error) {
	return secretbox.NewAEAD("isrvd-webssh", config.Current().Server.JWTSecret)
}

func sealSecret(aead cipher.AEAD, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return secretbox.Seal(aead, plain)
}

func openSecret(aead cipher.AEAD, value string) (string, error) {
	if !strings.HasPrefix(value, secretbox.Prefix) {
		return value, nil
	}
	plain, err := secretbox.Open(aead, value)
	if err != nil {
		return "", errors.New("SSH 认证信息解密失败，JWT 密钥可能已变更")
	}
	return plain, nil
}
