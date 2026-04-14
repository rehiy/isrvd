package swarm

import (
	"github.com/docker/docker/client"
)

// SwarmHandler Swarm 集群处理器
type SwarmHandler struct {
	dockerClient *client.Client
}

// NewSwarmHandler 创建 Swarm 处理器
func NewSwarmHandler(cli *client.Client) *SwarmHandler {
	return &SwarmHandler{dockerClient: cli}
}
