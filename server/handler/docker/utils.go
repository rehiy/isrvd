package docker

import (
	"archive/tar"
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
)

// formatPorts 格式化端口列表
func formatPorts(ports []types.Port) string {
	var result []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			result = append(result, strconv.Itoa(int(p.PublicPort))+"/"+p.Type+"->"+p.IP+":"+strconv.Itoa(int(p.PrivatePort)))
		} else {
			result = append(result, strconv.Itoa(int(p.PrivatePort))+"/"+p.Type)
		}
	}
	return strings.Join(result, ", ")
}

// parseDockerLogs 解析 Docker 日志格式
func parseDockerLogs(data []byte) []string {
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

// readAll 读取所有数据
func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// parseInt 解析整数
func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// min 返回最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
