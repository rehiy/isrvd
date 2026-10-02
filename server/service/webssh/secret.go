package webssh

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/rehiy/libgo/secure"

	"isrvd/server/config"
)

// SSH 密码/私钥轻量落盘加密：密钥由 JWT 密钥派生，仅防止文件被直接查看时泄露明文。
// 无前缀的值视为历史明文，下次保存时自动加密。
const sealedPrefix = "enc:"

func secretAEAD() (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("isrvd-webssh:" + config.Current().Server.JWTSecret))
	return secure.NewAESGCM(key[:])
}

func sealSecret(aead cipher.AEAD, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	raw, err := secure.AEADSeal(aead, []byte(plain), nil)
	if err != nil {
		return "", err
	}
	return sealedPrefix + base64.RawStdEncoding.EncodeToString(raw), nil
}

func openSecret(aead cipher.AEAD, value string) (string, error) {
	encoded, ok := strings.CutPrefix(value, sealedPrefix)
	if !ok {
		return value, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err == nil {
		plain, err := secure.AEADOpen(aead, raw, nil)
		if err == nil {
			return string(plain), nil
		}
	}

	return "", errors.New("SSH 认证信息解密失败，JWT 密钥可能已变更")
}
