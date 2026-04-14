package model

// ==================== Swarm 模型 ====================

// Swarm 节点信息
type SwarmNode struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Role          string `json:"role"`         // manager / worker
	Availability  string `json:"availability"` // active / pause / drain
	State         string `json:"state"`        // ready / down / unknown
	Addr          string `json:"addr"`
	EngineVersion string `json:"engineVersion"`
	Leader        bool   `json:"leader"`
}

// Swarm 服务信息
type SwarmService struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Image        string      `json:"image"`
	Mode         string      `json:"mode"` // replicated / global
	Replicas     *uint64     `json:"replicas"`
	RunningTasks int         `json:"runningTasks"`
	Ports        interface{} `json:"ports"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

// Swarm 任务信息
type SwarmTask struct {
	ID          string `json:"id"`
	ServiceID   string `json:"serviceID"`
	ServiceName string `json:"serviceName"`
	NodeID      string `json:"nodeID"`
	Slot        int    `json:"slot"`
	Image       string `json:"image"`
	State       string `json:"state"`
	Message     string `json:"message"`
	Err         string `json:"err"`
	UpdatedAt   string `json:"updatedAt"`
}

// Swarm 创建服务请求
type SwarmCreateServiceRequest struct {
	Name     string            `json:"name" binding:"required"`
	Image    string            `json:"image" binding:"required"`
	Mode     string            `json:"mode"`     // replicated | global
	Replicas int               `json:"replicas"` // mode=replicated 时有效
	Env      []string          `json:"env"`
	Args     []string          `json:"args"`
	Networks []string          `json:"networks"`
	Ports    []SwarmPortConfig `json:"ports"`
	Mounts   []SwarmMount      `json:"mounts"`
}

type SwarmPortConfig struct {
	Published int    `json:"published"`
	Target    int    `json:"target"`
	Protocol  string `json:"protocol"` // tcp | udp
}

type SwarmMount struct {
	Type   string `json:"type"` // bind | volume
	Source string `json:"source"`
	Target string `json:"target"`
}
