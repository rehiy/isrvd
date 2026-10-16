package node

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// 节点状态
const (
	StatusPending = "pending" // 待接入
	StatusActive  = "active"  // 可连接
	StatusRevoked = "revoked" // 已吊销，令牌失效
)

// ErrNotFound 表示节点不存在。
var ErrNotFound = errors.New("记录不存在")

// Node 节点记录。令牌仅保存 SHA-256 哈希，不保存明文或密文。
type Node struct {
	ID        string    `yaml:"id" json:"id"`                 // 节点 ID（自动生成）
	Name      string    `yaml:"name" json:"name"`             // 节点名称
	Status    string    `yaml:"status" json:"status"`         // pending / active / revoked
	Hostname  string    `yaml:"hostname" json:"hostname"`     // 注册时上报的主机名
	OS        string    `yaml:"os" json:"os"`                 // 注册时上报的操作系统
	Arch      string    `yaml:"arch" json:"arch"`             // 注册时上报的 CPU 架构
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`   // 创建时间
	TokenHash string    `yaml:"tokenHash,omitempty" json:"-"` // 节点令牌哈希
}

// NodeInfo 为接口提供节点数据及实时连接状态，不参与持久化。
type NodeInfo struct {
	Node
	Online       bool       `json:"online"`
	Latency      int64      `json:"latency"`
	AgentVersion string     `json:"agentVersion,omitempty"`
	Compatible   bool       `json:"compatible"`
	ConnectedAt  *time.Time `json:"connectedAt,omitempty"`
	RemoteAddr   string     `json:"remoteAddr,omitempty"`
}

// NodeRequest 节点创建与重命名请求
type NodeRequest struct {
	Name string `json:"name"` // 节点名称
}

// name 统一校验创建与重命名的节点名称。
func (r *NodeRequest) name() (string, error) {
	name := strings.TrimSpace(r.Name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return "", fmt.Errorf("节点名称长度须为 1–64 个字符")
	}
	return name, nil
}
