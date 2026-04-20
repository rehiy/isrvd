package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

// isWildcardIP 判断 IP 是否为通配地址（0.0.0.0 或 ::）
func isWildcardIP(ip string) bool {
	return ip == "" || ip == "0.0.0.0" || ip == "::"
}

// formatPorts 格式化端口列表：IPv4 优先、去重、通配地址省略 IP
//
// 算法：单次遍历，用 seen map 记录已输出的 key；
// IPv6 条目若已有对应 IPv4 条目则跳过，否则正常输出。
func formatPorts(ports []types.Port) []string {
	seen := make(map[string]bool, len(ports))
	result := make([]string, 0, len(ports))

	for _, p := range ports {
		var entry, key string
		isIPv6 := strings.Contains(p.IP, ":")

		if p.PublicPort > 0 {
			key = fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			// IPv6 且已有 IPv4 同 key，跳过
			if isIPv6 && seen[key] {
				continue
			}
			if isWildcardIP(p.IP) {
				entry = fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			} else {
				entry = fmt.Sprintf("%s:%d:%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
			}
		} else {
			key = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
			entry = key
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, entry)
		}
	}
	return result
}

// buildDockerfileTar 构建 Dockerfile 的 tar 包
func buildDockerfileTar(dockerfile string) (*bytes.Buffer, error) {
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	hdr := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return nil, err
	}
	tw.Close()
	return tarBuf, nil
}

// ParseDockerLogs 解析 Docker multiplexed stream 格式的日志数据
// 移除每帧前 8 字节的头部，返回纯文本行列表
func ParseDockerLogs(data []byte) []string {
	var logs []string
	for i := 0; i < len(data); {
		if i+8 > len(data) {
			break
		}
		size := int(data[i+4])<<24 | int(data[i+5])<<16 | int(data[i+6])<<8 | int(data[i+7])
		i += 8
		if i+size > len(data) || size <= 0 {
			break
		}
		logs = append(logs, string(data[i:i+size]))
		i += size
	}
	return logs
}

// registryHost 从仓库 URL 中提取 host 部分（去掉协议前缀和路径），用于拼接镜像引用
// 例如：https://csighub.tencentyun.com -> csighub.tencentyun.com
func registryHost(registryURL string) string {
	host := strings.TrimPrefix(registryURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	return host
}

// formatUnixTime 将 Unix 时间戳格式化为 RFC3339 字符串
func formatUnixTime(ts int64) string {
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
