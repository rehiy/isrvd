package docker

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/client"
)

// TLSConfig 远程 Docker daemon 的 TLS 连接参数，证书与私钥均为 PEM 文本。
type TLSConfig struct {
	SkipVerify bool   // 不校验服务端证书（仅用于测试环境）
	CA         string // 校验服务端证书的 CA；为空时使用系统根证书
	Cert       string // 客户端证书（双向认证，需与 Key 成对配置）
	Key        string // 客户端私钥
}

// Config 校验参数并构造 TLS 客户端配置。
// TLS 只对 tcp:// 地址有意义，host 为其他形式（unix://、npipe://、空）时返回错误。
func (c *TLSConfig) Config(host string) (*tls.Config, error) {
	if !strings.HasPrefix(strings.TrimSpace(host), "tcp://") {
		return nil, errors.New("启用 TLS 时 Docker Host 必须是 tcp:// 地址")
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: c.SkipVerify} //nolint:gosec // 由管理员显式选择

	if ca := strings.TrimSpace(c.CA); ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return nil, errors.New("CA 证书不是有效的 PEM 内容")
		}
		cfg.RootCAs = pool
	}

	cert, key := strings.TrimSpace(c.Cert), strings.TrimSpace(c.Key)
	switch {
	case cert == "" && key == "":
	case cert == "" || key == "":
		return nil, errors.New("客户端证书与私钥必须成对配置")
	default:
		pair, err := tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			return nil, fmt.Errorf("客户端证书或私钥无效: %w", err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	return cfg, nil
}

// withTLSConfig 将 TLS 配置应用到 SDK 客户端的传输层。
// SDK 自带的 WithTLSClientConfig 只接受文件路径，而这里的证书以 PEM 文本保存。
// 必须排在 WithHost 之后，且需要在 SDK 推导 scheme 前生效，才会改用 https。
func withTLSConfig(cfg *tls.Config) client.Opt {
	return func(c *client.Client) error {
		transport, ok := c.HTTPClient().Transport.(*http.Transport)
		if !ok {
			return fmt.Errorf("无法为传输层 %T 设置 TLS", c.HTTPClient().Transport)
		}
		transport.TLSClientConfig = cfg
		return nil
	}
}
