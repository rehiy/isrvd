// Package node 管理节点记录、token 认证和隧道会话，并提供受管机客户端。
package node

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/strutil"

	"isrvd/server/config"
)

var logger = logman.Named("node")

// Service 中控侧节点服务
type Service struct {
	nodes *nodeStore

	mu         sync.RWMutex
	conns      map[string]*nodeConn // 节点 ID → 当前在线会话
	connecting map[string]*Node     // 握手期间预占，防止并发连接
	closed     bool

	authLimit *rateLimiter // 令牌认证：按来源 IP 限制失败次数

	portmap   *config.PortMapConfig     // 启动时读取的 portmap 配置，修改配置文件后需重启生效
	listenMu  sync.Mutex                // 串行化端口转发的监听变更；加锁顺序：listenMu → mu
	listeners map[portKey]*portListener // 协议与端口 → 正在监听的转发规则
}

// NewService 创建中控侧节点服务；节点表保存在业务数据存储中（后端见 config.OpenData）
func NewService() (*Service, error) {
	nodes, err := newNodeStore()
	if err != nil {
		return nil, fmt.Errorf("初始化节点存储失败: %w", err)
	}
	s := &Service{
		nodes:      nodes,
		conns:      map[string]*nodeConn{},
		connecting: map[string]*Node{},
		authLimit:  &rateLimiter{hits: map[string][]time.Time{}},
		portmap:    config.Current().PortMap,
		listeners:  map[portKey]*portListener{},
	}
	s.portmapRestore()
	return s, nil
}

// Close 关闭全部在线会话与端口转发监听，应在应用退出或重载时调用
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	conns := s.conns
	s.conns = map[string]*nodeConn{}
	clear(s.connecting)
	s.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
	s.portmapStopWhere(func(*portListener) bool {
		return true
	})
}

// NodeList 列出全部节点，附带在线状态
func (s *Service) NodeList() []*NodeInfo {
	list := s.nodes.list()
	slices.SortFunc(list, func(a, b *Node) int { return a.CreatedAt.Compare(b.CreatedAt) })
	result := make([]*NodeInfo, len(list))
	for i, n := range list {
		result[i] = s.nodeInfo(n)
	}
	return result
}

// NodeCreateResult 节点创建结果，Token 仅在创建时返回，不持久化。
type NodeCreateResult struct {
	Node  *NodeInfo `json:"node"`
	Token string    `json:"token"`
}

// NodeCreate 创建待接入节点，令牌由密码学随机数采样为 86 个字母数字字符。
func (s *Service) NodeCreate(req *NodeRequest) (*NodeCreateResult, error) {
	name, err := req.name()
	if err != nil {
		return nil, err
	}
	token := strutil.Rand(86) // 86 个字母数字字符，熵约 512 位
	n := &Node{
		Name: name, Status: StatusPending,
		CreatedAt: time.Now(), TokenHash: hashToken(token),
	}
	if err := s.nodes.insert(n); err != nil {
		return nil, fmt.Errorf("创建节点失败: %w", err)
	}
	logger.Info("节点已创建", "id", n.ID, "name", n.Name)
	return &NodeCreateResult{Node: s.nodeInfo(n), Token: token}, nil
}

// NodeUpdate 重命名节点
func (s *Service) NodeUpdate(id string, req *NodeRequest) (*NodeInfo, error) {
	name, err := req.name()
	if err != nil {
		return nil, err
	}
	n, err := s.nodes.update(id, func(n *Node) error {
		n.Name = name
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("更新节点失败: %w", err)
	}
	logger.Info("节点已重命名", "id", id, "name", name)
	return s.nodeInfo(n), nil
}

// NodeRevoke 吊销节点：令牌立即失效并断开在线会话
func (s *Service) NodeRevoke(id string) error {
	err := s.dropNode(id, func() error {
		_, err := s.nodes.update(id, func(n *Node) error {
			n.Status, n.TokenHash, n.PortMaps = StatusRevoked, "", nil
			return nil
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("吊销节点失败: %w", err)
	}
	logger.Info("节点已吊销", "id", id)
	return nil
}

// NodeDelete 删除节点并断开在线会话
func (s *Service) NodeDelete(id string) error {
	err := s.dropNode(id, func() error {
		return s.nodes.remove(id)
	})
	if err != nil {
		return fmt.Errorf("删除节点失败: %w", err)
	}
	logger.Info("节点已删除", "id", id)
	return nil
}

// dropNode 在 s.mu 内执行 change（吊销或删除记录），成功后断开在线会话并撤销该节点的端口转发监听。
func (s *Service) dropNode(id string, change func() error) error {
	s.mu.Lock()
	if err := change(); err != nil {
		s.mu.Unlock()
		return err
	}
	c := s.conns[id]
	delete(s.conns, id)
	s.mu.Unlock()
	if c != nil {
		c.close()
	}
	s.portmapStopWhere(func(l *portListener) bool {
		return l.nodeID == id
	})
	return nil
}
