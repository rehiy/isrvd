package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// ShortID 返回 ID 的前 12 字符，不足 12 则返回原值
func ShortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
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
//
// daemon 位于其他主机时（Remote），只采用第 1 种：upperdir/workdir 是宿主机上带层哈希的路径，
// 不会与另一台主机的容器碰撞；而 IP、hostname 在不同主机间极易重复（默认网桥都是 172.17.0.x），
// 会把远程容器误判为自身，进而被禁止停止或删除。
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

	remote := s.Remote()
	if remote && len(selfMounts) == 0 {
		return "" // 没有可靠的识别依据，宁可不识别
	}

	hostname, _ := os.Hostname()

	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return ""
	}

	for _, ct := range containers {
		if len(selfMounts) > 0 {
			if info, err := s.client.ContainerInspect(ctx, ct.ID); err == nil {
				for _, key := range []string{"UpperDir", "WorkDir"} {
					if selfMounts[info.GraphDriver.Data[key]] {
						return ct.ID
					}
				}
			}
		}

		if remote {
			continue
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

// ─── 辅助函数 ───

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
