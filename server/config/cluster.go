package config

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidClusterConfig 表示多节点部署配置无效。
var ErrInvalidClusterConfig = errors.New("运行模式配置无效")

// ClusterConfig 是仅从环境变量读取的进程启动配置，不参与持久化或配置 API。
type ClusterConfig struct {
	Mode      string
	CenterURL string
	NodeToken string
}

var clusterConfig = ClusterConfig{Mode: "server"}

// Cluster 返回启动时确定的部署配置副本，修改环境变量后需重启。
func Cluster() ClusterConfig {
	return clusterConfig
}

func loadClusterEnv() error {
	cluster := ClusterConfig{
		Mode:      strings.ToLower(envOrDefault("ISRVD_MODE", "server")),
		CenterURL: strings.TrimRight(envOrDefault("ISRVD_CENTER_URL", ""), "/"),
		NodeToken: envOrDefault("ISRVD_NODE_TOKEN", ""),
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
	case "server", "center":
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
