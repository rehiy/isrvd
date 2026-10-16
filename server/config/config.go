package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Version 版本信息（编译时通过脚本注入）
var Version = "v0.0.0"

// ErrInvalidClusterConfig 表示多节点部署配置无效。
var ErrInvalidClusterConfig = errors.New("运行模式配置无效")

// SecretKeep 处理"留空保留原值"的密钥类字段：
// newVal 去除首尾空白后为空则返回 oldVal，否则返回去除首尾空白后的 newVal。
// 所有可由 API 写入且响应时不回显的密钥（JWT、OIDC、Copilot、APISIX、镜像仓库密码、Caddy 私钥等）
// 统一使用本函数，避免各处对空白的处理不一致。
func SecretKeep(newVal, oldVal string) string {
	newVal = strings.TrimSpace(newVal)
	if newVal == "" {
		return oldVal
	}
	return newVal
}

// ServerNormalize 填充 Server 默认值并归一化路径
func ServerNormalize(server *ServerConfig) *ServerConfig {
	if server == nil {
		server = &ServerConfig{}
	}
	if server.ListenAddr == "" {
		server.ListenAddr = ":8080"
	}
	if server.JWTExpiration == 0 {
		server.JWTExpiration = 86400
	}
	if server.MaxUploadSize == 0 {
		server.MaxUploadSize = 100 << 20
	}
	if server.RootDirectory == "" {
		server.RootDirectory = "."
	}
	if !filepath.IsAbs(server.RootDirectory) {
		if abs, err := filepath.Abs(server.RootDirectory); err == nil {
			server.RootDirectory = abs
		}
	}
	return server
}

// ClusterNormalize 填充多节点部署默认值并归一化连接信息
func ClusterNormalize(cluster *ClusterConfig) *ClusterConfig {
	if cluster == nil {
		cluster = &ClusterConfig{}
	}
	cluster.Mode = strings.ToLower(strings.TrimSpace(cluster.Mode))
	if cluster.Mode == "" {
		cluster.Mode = "server"
	}
	cluster.CenterURL = strings.TrimRight(strings.TrimSpace(cluster.CenterURL), "/")
	cluster.NodeToken = strings.TrimSpace(cluster.NodeToken)
	return cluster
}

var nodeIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ClusterValidate 统一校验启动及控制台提交的多节点部署配置，不回显连接凭据。
func ClusterValidate(cluster *ClusterConfig) error {
	switch cluster.Mode {
	case "server", "center":
		return nil
	case "agent":
		u, err := parseHTTPURL("cluster.centerUrl", cluster.CenterURL)
		if err != nil {
			return fmt.Errorf("%w：%v", ErrInvalidClusterConfig, err)
		}
		if u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return fmt.Errorf("%w：服务器地址不能包含路径、查询参数或片段", ErrInvalidClusterConfig)
		}
		id, secret, ok := strings.Cut(cluster.NodeToken, ".")
		decoded, err := base64.RawURLEncoding.DecodeString(secret)
		if !ok || !nodeIDPattern.MatchString(id) || err != nil || len(decoded) != 32 {
			return fmt.Errorf("%w：请填写中控生成的有效连接 token", ErrInvalidClusterConfig)
		}
		return nil
	default:
		return fmt.Errorf("%w：mode 只能是 server、center 或 agent", ErrInvalidClusterConfig)
	}
}

// THANormalize 填充 THA 默认值
func THANormalize(tha *THAConfig) *THAConfig {
	if tha == nil {
		tha = &THAConfig{}
	}
	if tha.HeaderName == "" {
		tha.HeaderName = "X-Username"
	}
	// 未配置可信代理来源时默认仅信任本机，避免可直连服务端口的客户端伪造用户名 Header
	if len(tha.TrustedCIDRs) == 0 {
		tha.TrustedCIDRs = []string{"127.0.0.1/32", "::1/128"}
	}
	return tha
}

// OIDCNormalize 填充 OIDC 默认值
func OIDCNormalize(oidc *OIDCConfig) *OIDCConfig {
	if oidc == nil {
		oidc = &OIDCConfig{}
	}
	if oidc.UsernameClaim == "" {
		oidc.UsernameClaim = "sub"
	}
	if len(oidc.Scopes) == 0 {
		oidc.Scopes = []string{"openid", "profile", "email"}
	}
	return oidc
}

// PasskeyNormalize 填充 Passkey 默认值
func PasskeyNormalize(passkey *PasskeyConfig) *PasskeyConfig {
	if passkey == nil {
		passkey = &PasskeyConfig{}
	}
	if passkey.Timeout == 0 {
		passkey.Timeout = 60000
	}
	return passkey
}

// PasswordNormalize 填充 Password 默认值
func PasswordNormalize(password *PasswordConfig) *PasswordConfig {
	if password == nil {
		password = &PasswordConfig{}
	}
	if password.MinLength == 0 {
		password.MinLength = 6
	}
	return password
}

// MonitorNormalize 填充 Monitor 默认值
func MonitorNormalize(monitor *MonitorConfig) *MonitorConfig {
	if monitor == nil {
		monitor = &MonitorConfig{}
	}
	switch monitor.Interval {
	case 5, 15, 30, 60:
	default:
		monitor.Interval = 0
	}
	return monitor
}

// FaultAlertNormalize 填充默认阈值；不自动开启告警。
func FaultAlertNormalize(events *FaultAlertConfig) FaultAlertConfig {
	var result FaultAlertConfig
	if events != nil {
		result = *events
	}
	if result.RestartThreshold <= 0 || result.RestartThreshold > 10000 {
		result.RestartThreshold = 3
	}
	if result.RestartWindow < 30 || result.RestartWindow > 86400 {
		result.RestartWindow = 300
	}
	if result.CertificateDays <= 0 || result.CertificateDays > 3650 {
		result.CertificateDays = 14
	}
	return result
}

// NotifyNormalize 保证应用故障配置完整，并保留旧配置的关闭状态。
func NotifyNormalize(value *NotifyConfig) *NotifyConfig {
	result := &NotifyConfig{}
	if value != nil {
		*result = *value
	}
	events := FaultAlertNormalize(result.Events)
	result.Events = &events
	return result
}
