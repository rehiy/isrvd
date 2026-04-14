package swarm

// SwarmNode Swarm 节点信息
type SwarmNode struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Role          string `json:"role"`
	Availability  string `json:"availability"`
	State         string `json:"state"`
	Addr          string `json:"addr"`
	EngineVersion string `json:"engineVersion"`
	Leader        bool   `json:"leader"`
}

// SwarmService Swarm 服务信息
type SwarmService struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Image        string      `json:"image"`
	Mode         string      `json:"mode"`
	Replicas     *uint64     `json:"replicas"`
	RunningTasks int         `json:"runningTasks"`
	Ports        interface{} `json:"ports"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

// SwarmTask Swarm 任务信息
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

// SwarmCreateServiceRequest Swarm 创建服务请求
type SwarmCreateServiceRequest struct {
	Name     string            `json:"name" binding:"required"`
	Image    string            `json:"image" binding:"required"`
	Mode     string            `json:"mode"`
	Replicas int               `json:"replicas"`
	Env      []string          `json:"env"`
	Args     []string          `json:"args"`
	Networks []string          `json:"networks"`
	Ports    []SwarmPortConfig `json:"ports"`
	Mounts   []SwarmMount      `json:"mounts"`
}

// SwarmPortConfig Swarm 端口配置
type SwarmPortConfig struct {
	Published int    `json:"published"`
	Target    int    `json:"target"`
	Protocol  string `json:"protocol"`
}

// SwarmMount Swarm 挂载配置
type SwarmMount struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}
