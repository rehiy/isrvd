package webssh

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"isrvd/server/config"
)

// SSH 密码/私钥轻量落盘加密：密钥由 JWT 密钥派生，仅防止文件被直接查看时泄露明文。
// 无前缀的值视为历史明文，下次保存时自动加密。
const sealedPrefix = "enc:"

func secretAEAD() (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("isrvd-webssh:" + config.Current().Server.JWTSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func sealSecret(aead cipher.AEAD, plain string) string {
	if plain == "" {
		return ""
	}
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)
	return sealedPrefix + base64.RawStdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plain), nil))
}

func openSecret(aead cipher.AEAD, value string) (string, error) {
	encoded, ok := strings.CutPrefix(value, sealedPrefix)
	if !ok {
		return value, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	n := aead.NonceSize()
	if err == nil && len(raw) >= n {
		var plain []byte
		if plain, err = aead.Open(nil, raw[:n], raw[n:], nil); err == nil {
			return string(plain), nil
		}
	}
	return "", errors.New("SSH 认证信息解密失败，JWT 密钥可能已变更")
}
