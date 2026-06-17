package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// ─── 辅助函数 ───

// ShortID 返回 ID 的前 12 字符，不足 12 则返回原值
func ShortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ParseDockerLogs 解析 Docker multiplexed stream 格式的日志数据。
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

// formatPorts 格式化端口列表：IPv4 优先、去重、通配地址省略 IP
//
// 算法：单次遍历，用 seen map 记录已输出的 key；
// IPv6 条目若已有对应 IPv4 条目则跳过，否则正常输出。
func formatPorts(ports []container.Port) []string {
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
			if p.IP == "" || p.IP == "0.0.0.0" || p.IP == "::" {
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

// consumeImageStream 消费 Docker 镜像操作的 JSON 流，返回最后一条 status 消息。
// 遇到流中 error 字段时立即返回错误。
func consumeImageStream(dec *json.Decoder) (string, error) {
	var lastMessage string
	for {
		var msg struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			return "", errors.New(msg.Error)
		}
		if msg.Status != "" {
			lastMessage = msg.Status
		}
	}
	return lastMessage, nil
}

// SelfContainerID 获取并缓存当前容器的完整 ID。如果不在容器中，返回空字符串。
func (s *DockerService) SelfContainerID(ctx context.Context) string {
	s.selfIDOnce.Do(func() {
		s.selfID = s.resolveSelfContainerID(ctx)
	})
	return s.selfID
}

// resolveSelfContainerID 解析当前运行的自身容器 ID。
// 识别链条：
// 1. 优先使用 mountinfo 匹配 overlay 存储驱动的 upperdir/workdir (最精准，兼容 host 网络)。
// 2. 降级使用 IP 匹配网络端点 (兼容 btrfs/zfs 等非 overlay 存储驱动)。
// 3. 降级使用 hostname 兜底。
func (s *DockerService) resolveSelfContainerID(ctx context.Context) string {
	selfMounts := make(map[string]bool)
	if data, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) != 2 {
				continue
			}
			pre := strings.Fields(parts[0])
			post := strings.Fields(parts[1])
			if len(pre) < 5 || pre[4] != "/" || len(post) < 3 || post[0] != "overlay" {
				continue
			}
			for opt := range strings.SplitSeq(post[2], ",") {
				if v, ok := strings.CutPrefix(opt, "upperdir="); ok {
					selfMounts[v] = true
				}
				if v, ok := strings.CutPrefix(opt, "workdir="); ok {
					selfMounts[v] = true
				}
			}
		}
	}

	selfIPs := make(map[string]bool)
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok {
						selfIPs[ipNet.IP.String()] = true
					}
				}
			}
		}
	}

	hostname, _ := os.Hostname()

	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return ""
	}

	for _, ct := range containers {
		if len(selfMounts) > 0 {
			if info, err := s.client.ContainerInspect(ctx, ct.ID); err == nil {
				for _, key := range []string{"UpperDir", "WorkDir", "MergedDir"} {
					if selfMounts[info.GraphDriver.Data[key]] {
						return ct.ID
					}
				}
			}
		}

		hasIP := false
		if ct.NetworkSettings != nil {
			for _, ep := range ct.NetworkSettings.Networks {
				if ep == nil || ep.IPAddress == "" {
					continue
				}
				hasIP = true
				if selfIPs[ep.IPAddress] {
					return ct.ID
				}
			}
		}

		if !hasIP {
			if hostname != "" && strings.HasPrefix(hostname, ShortID(ct.ID)) {
				return ct.ID
			}
		}
	}
	return ""
}
