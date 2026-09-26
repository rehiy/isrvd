package certutil

import (
	"crypto/x509"
	"encoding/pem"
)

// PEMParse 解析第一个 CERTIFICATE block，跳过私钥等其他 block。
// 内容缺失或证书无效时返回 nil。
func PEMParse(data []byte) *x509.Certificate {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil
		}
		if block.Type == "CERTIFICATE" {
			cert, _ := x509.ParseCertificate(block.Bytes)
			return cert
		}
	}
	return nil
}
