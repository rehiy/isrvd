package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ErrInvalidClusterConfig 表示多节点部署配置无效。
var ErrInvalidClusterConfig = errors.New("运行模式配置无效")

// ClusterConfig 是仅从环境变量读取的进程启动配置，不参与持久化或配置 API。
type ClusterConfig struct {
	Mode      string
	CenterURL string
	NodeToken string

	// 以下仅 center 使用：端口转发的监听地址与允许分配的监听端口范围，范围为 0 表示未启用。
	ForwardBind    string
	ForwardPortMin int
	ForwardPortMax int
}

var clusterConfig = ClusterConfig{Mode: "server"}

// Cluster 返回启动时确定的部署配置副本，修改环境变量后需重启。
func Cluster() ClusterConfig {
	return clusterConfig
}

// ForwardEnabled 判断中控是否启用端口转发。
func (c ClusterConfig) ForwardEnabled() bool {
	return c.ForwardPortMax > 0
}

// ForwardPortAllowed 判断端口是否在允许分配的监听范围内。
func (c ClusterConfig) ForwardPortAllowed(port int) bool {
	return c.ForwardEnabled() && port >= c.ForwardPortMin && port <= c.ForwardPortMax
}

func loadClusterEnv() error {
	cluster := ClusterConfig{
		Mode:        strings.ToLower(envOrDefault("ISRVD_MODE", "server")),
		CenterURL:   strings.TrimRight(envOrDefault("ISRVD_CENTER_URL", ""), "/"),
		NodeToken:   envOrDefault("ISRVD_NODE_TOKEN", ""),
		ForwardBind: strings.TrimSpace(envOrDefault("ISRVD_FORWARD_BIND", "127.0.0.1")),
	}
	if cluster.Mode == "center" {
		var err error
		if cluster.ForwardPortMin, cluster.ForwardPortMax, err = parsePortRange(envOrDefault("ISRVD_FORWARD_PORTS", "")); err != nil {
			return fmt.Errorf("%w：ISRVD_FORWARD_PORTS %v", ErrInvalidClusterConfig, err)
		}
	}
	if err := validateCluster(&cluster); err != nil {
		return err
	}
	clusterConfig = cluster
	return nil
}

// validateCluster 校验环境变量中的多节点部署配置，不回显连接凭据。
func validateCluster(cluster *ClusterConfig) error {
	switch cluster.Mode {
	case "server":
		return nil
	case "center":
		if cluster.ForwardEnabled() && net.ParseIP(cluster.ForwardBind) == nil {
			return fmt.Errorf("%w：ISRVD_FORWARD_BIND 必须是 IP 地址", ErrInvalidClusterConfig)
		}
		return nil
	case "agent":
		u, err := parseHTTPURL("ISRVD_CENTER_URL", cluster.CenterURL)
		if err != nil {
			return fmt.Errorf("%w：%v", ErrInvalidClusterConfig, err)
		}
		if u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return fmt.Errorf("%w：服务器地址不能包含路径、查询参数或片段", ErrInvalidClusterConfig)
		}
		if len(cluster.NodeToken) < 64 || len(cluster.NodeToken) > 96 || strings.ContainsAny(cluster.NodeToken, " \t\r\n") {
			return fmt.Errorf("%w：请填写中控生成的有效连接 token", ErrInvalidClusterConfig)
		}
		return nil
	default:
		return fmt.Errorf("%w：mode 只能是 server、center 或 agent", ErrInvalidClusterConfig)
	}
}

// parsePortRange 解析 "20000-20999" 或单个端口 "20022"；空串表示未启用。
func parsePortRange(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, nil
	}
	lo, hi, found := strings.Cut(value, "-")
	if !found {
		hi = lo
	}
	min, err1 := strconv.Atoi(strings.TrimSpace(lo))
	max, err2 := strconv.Atoi(strings.TrimSpace(hi))
	if err1 != nil || err2 != nil || min < 1 || max > 65535 || min > max {
		return 0, 0, fmt.Errorf("格式应为 20000-20999 或单个端口（1–65535）")
	}
	return min, max, nil
}
