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

// SwarmNodeInspect 节点详情
type SwarmNodeInspect struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	Role          string            `json:"role"`
	Availability  string            `json:"availability"`
	State         string            `json:"state"`
	Addr          string            `json:"addr"`
	EngineVersion string            `json:"engineVersion"`
	Leader        bool              `json:"leader"`
	OS            string            `json:"os"`
	Architecture  string            `json:"architecture"`
	CPUs          int64             `json:"cpus"`
	MemoryBytes   int64             `json:"memoryBytes"`
	Labels        map[string]string `json:"labels"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

// SwarmServicePort 服务端口信息
type SwarmServicePort struct {
	Protocol      string `json:"protocol"`
	TargetPort    uint32 `json:"targetPort"`
	PublishedPort uint32 `json:"publishedPort"`
	PublishMode   string `json:"publishMode"`
}

// SwarmServiceMount 服务挂载信息
type SwarmServiceMount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// SwarmServiceInspect 服务详情
type SwarmServiceInspect struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Mode         string              `json:"mode"`
	Replicas     *uint64             `json:"replicas"`
	RunningTasks int                 `json:"runningTasks"`
	Ports        []SwarmServicePort  `json:"ports"`
	Env          []string            `json:"env"`
	Args         []string            `json:"args"`
	Networks     []string            `json:"networks"`
	Mounts       []SwarmServiceMount `json:"mounts"`
	Labels       map[string]string   `json:"labels"`
	Constraints  []string            `json:"constraints"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
}
