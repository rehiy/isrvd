package helper

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/rehiy/libgo/secure"
)

// 落盘用的轻量加密：AES-GCM 加密后以 enc: 前缀的 base64 文本保存。
//
// 密钥由「用途标签 + 密钥材料（通常是 JWT 密钥）」派生，仅防止文件被直接查看时泄露明文；
// 不同标签派生出不同的密钥，互不通用。

// SecretPrefix 密文的统一前缀，用于区分密文与历史明文
const SecretPrefix = "enc:"

// ErrSecretNotSealed 值没有密文前缀，或密文编码已损坏
var ErrSecretNotSealed = errors.New("值未加密或已损坏")

// NewSecretAEAD 按用途标签与密钥材料派生加密器
func NewSecretAEAD(label, material string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(label + ":" + material))
	return secure.NewAESGCM(key[:])
}

// SealSecret 加密明文，返回带 SecretPrefix 的密文
func SealSecret(aead cipher.AEAD, plain string) (string, error) {
	raw, err := secure.AEADSeal(aead, []byte(plain), nil)
	if err != nil {
		return "", err
	}
	return SecretPrefix + base64.RawStdEncoding.EncodeToString(raw), nil
}

// OpenSecret 解密 SealSecret 的结果；无前缀或编码损坏返回 ErrSecretNotSealed，密钥不符返回解密错误
func OpenSecret(aead cipher.AEAD, value string) (string, error) {
	encoded, ok := strings.CutPrefix(value, SecretPrefix)
	if !ok {
		return "", ErrSecretNotSealed
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrSecretNotSealed
	}
	plain, err := secure.AEADOpen(aead, raw, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
