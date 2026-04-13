package model

// Docker 容器信息
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Ports   string            `json:"ports"`
	Created int64             `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// Docker 镜像信息
type ImageInfo struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"shortId"`
	RepoTags []string `json:"repoTags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
}

// Docker 网络信息
type NetworkInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Driver  string   `json:"driver"`
	Subnet  string   `json:"subnet"`
	Scope   string   `json:"scope"`
	Network []string `json:"networks,omitempty"`
}

// Docker 卷信息
type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"createdAt"`
	Size       int64  `json:"size"`
}

// Docker 信息概览
type DockerInfo struct {
	ContainersRunning int64 `json:"containersRunning"`
	ContainersStopped int64 `json:"containersStopped"`
	ContainersPaused  int64 `json:"containersPaused"`
	ImagesTotal       int64 `json:"imagesTotal"`
	VolumesTotal      int64 `json:"volumesTotal"`
	NetworksTotal     int64 `json:"networksTotal"`
}

// ==================== 请求模型 ====================

// 容器操作请求
type ContainerActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // start, stop, restart, remove, pause, unpause
}

// 创建容器请求
type ContainerCreateRequest struct {
	Image      string            `json:"image" binding:"required"`
	Name       string            `json:"name"`
	Cmd        []string          `json:"cmd"`
	Env        []string          `json:"env"`
	Ports      map[string]string `json:"ports"`      // e.g., {"8080": "80"}
	Volumes    []VolumeMapping   `json:"volumes"`    // 目录映射
	Network    string            `json:"network"`    // 网络模式: bridge, host, none
	Restart    string            `json:"restart"`    // 重启策略: no, always, on-failure, unless-stopped
	Memory     int64             `json:"memory"`     // 内存限制 (MB)
	Cpus       float64           `json:"cpus"`       // CPU 限制 (核心数)
	Workdir    string            `json:"workdir"`    // 工作目录
	User       string            `json:"user"`       // 用户
	Hostname   string            `json:"hostname"`   // 主机名
	Privileged bool              `json:"privileged"` // 特权模式
	CapAdd     []string          `json:"capAdd"`     // 添加 Linux 能力
	CapDrop    []string          `json:"capDrop"`    // 移除 Linux 能力
}

// 目录映射
type VolumeMapping struct {
	HostPath      string `json:"hostPath"`      // 主机路径
	ContainerPath string `json:"containerPath"` // 容器路径
	ReadOnly      bool   `json:"readOnly"`      // 只读
}

// 镜像操作请求
type ImageActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // remove
}

// 拉取镜像请求
type ImagePullRequest struct {
	Image string `json:"image" binding:"required"`
	Tag   string `json:"tag"` // 默认 latest
}

// 网络操作请求
type NetworkActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // remove
}

// 创建网络请求
type NetworkCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"` // 默认 bridge
	Subnet string `json:"subnet"`
}

// 卷操作请求
type VolumeActionRequest struct {
	Name   string `json:"name" binding:"required"`
	Action string `json:"action" binding:"required"` // remove
}

// 创建卷请求
type VolumeCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"` // 默认 local
}

// 日志请求
type ContainerLogsRequest struct {
	ID     string `json:"id" binding:"required"`
	Tail   string `json:"tail"`   // 默认 "100"
	Follow bool   `json:"follow"` // 是否跟随
}

// 容器统计信息请求
type ContainerStatsRequest struct {
	ID string `json:"id" binding:"required"`
}

// 容器统计信息响应
type ContainerStatsResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpuPercent"`
	CpuCores      int     `json:"cpuCores"`
	CpuFreq       float64 `json:"cpuFreq"`
	MemoryUsage   int64   `json:"memoryUsage"`
	MemoryLimit   int64   `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     int64   `json:"networkRx"`
	NetworkTx     int64   `json:"networkTx"`
	BlockRead     int64   `json:"blockRead"`
	BlockWrite    int64   `json:"blockWrite"`
	Pids          int64   `json:"pids"`
	// 新增指标
	PidsLimit     int64                 `json:"pidsLimit"`     // 进程数限制
	CpuThrottled  *CpuThrottledData     `json:"cpuThrottled"`  // CPU 节流数据
	NetworkDetail map[string]*NetDetail `json:"networkDetail"` // 网卡详细统计
	BlockDetail   []*BlockDetail        `json:"blockDetail"`   // 磁盘设备详细统计
	ProcessList   *ContainerProcessList `json:"processList"`   // 进程列表
}

// CPU 节流数据
type CpuThrottledData struct {
	Periods          uint64 `json:"periods"`          // 总调度周期数
	ThrottledPeriods uint64 `json:"throttledPeriods"` // 被节流的周期数
	ThrottledTime    uint64 `json:"throttledTime"`    // 被节流的总时间（纳秒）
}

// 网卡详细统计
type NetDetail struct {
	RxBytes   uint64 `json:"rxBytes"`
	RxPackets uint64 `json:"rxPackets"`
	RxErrors  uint64 `json:"rxErrors"`
	RxDropped uint64 `json:"rxDropped"`
	TxBytes   uint64 `json:"txBytes"`
	TxPackets uint64 `json:"txPackets"`
	TxErrors  uint64 `json:"txErrors"`
	TxDropped uint64 `json:"txDropped"`
}

// 磁盘设备详细统计
type BlockDetail struct {
	Major uint64 `json:"major"` // 主设备号
	Minor uint64 `json:"minor"` // 次设备号
	Read  uint64 `json:"read"`  // 读取字节数
	Write uint64 `json:"write"` // 写入字节数
}

// 容器进程列表
type ContainerProcessList struct {
	Titles    []string   `json:"titles"`    // 表头
	Processes [][]string `json:"processes"` // 进程信息行
}

// 网络中的容器信息
type NetworkContainerInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
	MacAddress string `json:"macAddress"`
}

// 网络详情响应
type NetworkInspectResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Driver     string                  `json:"driver"`
	Scope      string                  `json:"scope"`
	Subnet     string                  `json:"subnet"`
	Gateway    string                  `json:"gateway"`
	Internal   bool                    `json:"internal"`
	EnableIPv6 bool                    `json:"enableIPv6"`
	Containers []*NetworkContainerInfo `json:"containers"`
}

// 使用卷的容器信息
type VolumeUsedByContainer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

// 数据卷详情响应
type VolumeInspectResponse struct {
	Name       string                   `json:"name"`
	Driver     string                   `json:"driver"`
	Mountpoint string                   `json:"mountpoint"`
	CreatedAt  string                   `json:"createdAt"`
	Scope      string                   `json:"scope"`
	Size       int64                    `json:"size"`
	RefCount   int64                    `json:"refCount"`
	UsedBy     []*VolumeUsedByContainer `json:"usedBy"`
}

// 镜像标签请求
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`
	RepoTag string `json:"repoTag" binding:"required"` // 新标签，如: myimage:v1
}

// 镜像搜索请求
type ImageSearchRequest struct {
	Term string `json:"term" binding:"required"` // 搜索关键词
}

// 镜像搜索结果
type ImageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsOfficial  bool   `json:"isOfficial"`
	IsAutomated bool   `json:"isAutomated"`
	StarCount   int    `json:"starCount"`
}

// 镜像构建请求
type ImageBuildRequest struct {
	Dockerfile string `json:"dockerfile" binding:"required"` // Dockerfile 内容
	Tag        string `json:"tag"`                           // 镜像标签，如: myimage:v1
}

// 容器配置响应（从 compose 文件读取）
type ContainerConfigResponse struct {
	Image      string            `json:"image"`
	Name       string            `json:"name"`
	Cmd        []string          `json:"cmd,omitempty"`
	Env        []string          `json:"env,omitempty"`
	Ports      map[string]string `json:"ports,omitempty"`
	Volumes    []VolumeMapping   `json:"volumes,omitempty"`
	Network    string            `json:"network,omitempty"`
	Restart    string            `json:"restart,omitempty"`
	Memory     int64             `json:"memory,omitempty"`
	Cpus       float64           `json:"cpus,omitempty"`
	Workdir    string            `json:"workdir,omitempty"`
	User       string            `json:"user,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Privileged bool              `json:"privileged,omitempty"`
	CapAdd     []string          `json:"capAdd,omitempty"`
	CapDrop    []string          `json:"capDrop,omitempty"`
}

// ==================== 镜像仓库模型 ====================

// 镜像仓库信息
type RegistryInfo struct {
	Name     string `json:"name"`     // 仓库名称
	URL      string `json:"url"`      // 仓库地址
	Username string `json:"username"` // 用户名
}

// 从仓库拉取镜像请求
type ImagePullFromRegistryRequest struct {
	Image       string `json:"image" binding:"required"`       // 仓库中的镜像名称，如 myapp:latest
	RegistryURL string `json:"registryUrl" binding:"required"` // 源仓库地址
}

// 镜像推送请求
type ImagePushRequest struct {
	Image       string `json:"image" binding:"required"`       // 镜像名称（含标签）
	RegistryURL string `json:"registryUrl" binding:"required"` // 目标仓库地址
	Namespace   string `json:"namespace"`                      // 目标命名空间（可选，如 myteam）
}

// 容器配置更新请求
type ContainerUpdateRequest struct {
	Name       string            `json:"name" binding:"required"`
	Image      string            `json:"image" binding:"required"`
	Cmd        []string          `json:"cmd"`
	Env        []string          `json:"env"`
	Ports      map[string]string `json:"ports"`
	Volumes    []VolumeMapping   `json:"volumes"`
	Network    string            `json:"network"`
	Restart    string            `json:"restart"`
	Memory     int64             `json:"memory"`
	Cpus       float64           `json:"cpus"`
	Workdir    string            `json:"workdir"`
	User       string            `json:"user"`
	Hostname   string            `json:"hostname"`
	Privileged bool              `json:"privileged"`
	CapAdd     []string          `json:"capAdd"`
	CapDrop    []string          `json:"capDrop"`
}
