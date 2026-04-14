package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
)

// parseInt 解析整数
func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// formatPorts 格式化端口列表
func formatPorts(ports []types.Port) []string {
	var result []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			result = append(result, fmt.Sprintf("%d/%s->%s:%d", p.PublicPort, p.Type, p.IP, p.PrivatePort))
		} else {
			result = append(result, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
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
