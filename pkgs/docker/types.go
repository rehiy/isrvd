package docker

// DockerConfig Docker 配置（由外部注入，解除对 server/config 的依赖）
type DockerConfig struct {
	Host          string            // Docker 连接地址
	ContainerRoot string            // 容器数据根目录
	Registries    []*RegistryConfig // 镜像仓库配置列表
}

// RegistryConfig 镜像仓库配置
type RegistryConfig struct {
	Name        string // 仓库名称
	URL         string // 仓库地址
	Username    string // 用户名
	Password    string // 密码
	Description string // 仓库描述
}

// ContainerInfo Docker 容器信息
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Ports   []string          `json:"ports"`
	Created int64             `json:"created"`
	IsSwarm bool              `json:"isSwarm,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// ImageInfo Docker 镜像信息
type ImageInfo struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"shortId"`
	RepoTags []string `json:"repoTags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
}

// ImageLayerInfo 镜像层信息
type ImageLayerInfo struct {
	Digest    string `json:"digest"`    // 层 digest（sha256:...）
	CreatedBy string `json:"createdBy"` // 构建命令
	Created   string `json:"created"`   // 创建时间
	Size      int64  `json:"size"`      // 层大小（字节），-1 表示空层
	Empty     bool   `json:"empty"`     // 是否为空层（无文件变更）
}

// ImageInspectResponse 镜像详情响应
type ImageInspectResponse struct {
	ID           string            `json:"id"`
	ShortID      string            `json:"shortId"`
	RepoTags     []string          `json:"repoTags"`
	RepoDigests  []string          `json:"repoDigests"`
	Size         int64             `json:"size"`
	VirtualSize  int64             `json:"virtualSize"`
	Created      string            `json:"created"`
	Author       string            `json:"author"`
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	Cmd          []string          `json:"cmd"`
	Entrypoint   []string          `json:"entrypoint"`
	Env          []string          `json:"env"`
	WorkingDir   string            `json:"workingDir"`
	ExposedPorts []string          `json:"exposedPorts"`
	Labels       map[string]string `json:"labels"`
	Layers       int               `json:"layers"`
	LayerDetails []*ImageLayerInfo `json:"layerDetails"`
}

// NetworkInfo Docker 网络信息
type NetworkInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Driver  string   `json:"driver"`
	Subnet  string   `json:"subnet"`
	Scope   string   `json:"scope"`
	Network []string `json:"networks,omitempty"`
}

// VolumeInfo Docker 卷信息
type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"createdAt"`
	Size       int64  `json:"size"`
}

// DockerInfo Docker 信息概览
type DockerInfo struct {
	ContainersRunning  int64    `json:"containersRunning"`
	ContainersStopped  int64    `json:"containersStopped"`
	ContainersPaused   int64    `json:"containersPaused"`
	ImagesTotal        int64    `json:"imagesTotal"`
	VolumesTotal       int64    `json:"volumesTotal"`
	NetworksTotal      int64    `json:"networksTotal"`
	RegistryMirrors    []string `json:"registryMirrors"`
	IndexServerAddress string   `json:"indexServerAddress"`
}

// ContainerActionRequest 容器操作请求
type ContainerActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// ContainerCreateRequest 创建容器请求
type ContainerCreateRequest struct {
	Image      string            `json:"image" binding:"required"`
	Name       string            `json:"name"`
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

// ContainerUpdateRequest 容器配置更新请求
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

// VolumeMapping 目录映射
type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

// ImageActionRequest 镜像操作请求
type ImageActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// ImagePullRequest 拉取镜像请求
type ImagePullRequest struct {
	Image string `json:"image" binding:"required"`
	Tag   string `json:"tag"`
}

// ImageTagRequest 镜像标签请求
type ImageTagRequest struct {
	ID      string `json:"id" binding:"required"`
	RepoTag string `json:"repoTag" binding:"required"`
}

// ImageSearchResult 镜像搜索结果
type ImageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsOfficial  bool   `json:"isOfficial"`
	IsAutomated bool   `json:"isAutomated"`
	StarCount   int    `json:"starCount"`
}

// ImageBuildRequest 镜像构建请求
type ImageBuildRequest struct {
	Dockerfile string `json:"dockerfile" binding:"required"`
	Tag        string `json:"tag"`
}

// NetworkActionRequest 网络操作请求
type NetworkActionRequest struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// NetworkCreateRequest 创建网络请求
type NetworkCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

// NetworkContainerInfo 网络中的容器信息
type NetworkContainerInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
	MacAddress string `json:"macAddress"`
}

// NetworkInspectResponse 网络详情响应
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

// VolumeActionRequest 卷操作请求
type VolumeActionRequest struct {
	Name   string `json:"name" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// VolumeCreateRequest 创建卷请求
type VolumeCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"`
}

// VolumeUsedByContainer 使用卷的容器信息
type VolumeUsedByContainer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

// VolumeInspectResponse 数据卷详情响应
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

// ContainerLogsRequest 日志请求
type ContainerLogsRequest struct {
	ID     string `json:"id" binding:"required"`
	Tail   string `json:"tail"`
	Follow bool   `json:"follow"`
}

// ContainerStatsResponse 容器统计信息响应
type ContainerStatsResponse struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	CPUPercent    float64               `json:"cpuPercent"`
	CPUCores      int                   `json:"cpuCores"`
	CPUFreq       float64               `json:"cpuFreq"`
	MemoryUsage   int64                 `json:"memoryUsage"`
	MemoryLimit   int64                 `json:"memoryLimit"`
	MemoryPercent float64               `json:"memoryPercent"`
	NetworkRx     int64                 `json:"networkRx"`
	NetworkTx     int64                 `json:"networkTx"`
	BlockRead     int64                 `json:"blockRead"`
	BlockWrite    int64                 `json:"blockWrite"`
	Pids          int64                 `json:"pids"`
	PidsLimit     int64                 `json:"pidsLimit"`
	CPUThrottled  *CPUThrottledData     `json:"cpuThrottled"`
	NetworkDetail map[string]*NetDetail `json:"networkDetail"`
	BlockDetail   []*BlockDetail        `json:"blockDetail"`
	ProcessList   *ContainerProcessList `json:"processList"`
}

// CPUThrottledData CPU 节流数据
type CPUThrottledData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttledPeriods"`
	ThrottledTime    uint64 `json:"throttledTime"`
}

// NetDetail 网卡详细统计
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

// BlockDetail 磁盘设备详细统计
type BlockDetail struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Read  uint64 `json:"read"`
	Write uint64 `json:"write"`
}

// ContainerProcessList 容器进程列表
type ContainerProcessList struct {
	Titles    []string   `json:"titles"`
	Processes [][]string `json:"processes"`
}

// ContainerConfigResponse 容器配置响应（从 compose 文件读取）
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

// RegistryInfo 镜像仓库信息
type RegistryInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Description string `json:"description"`
}

// ImagePullFromRegistryRequest 从仓库拉取镜像请求
type ImagePullFromRegistryRequest struct {
	Image       string `json:"image" binding:"required"`
	RegistryURL string `json:"registryUrl" binding:"required"`
	Namespace   string `json:"namespace"`
}

// ImagePushRequest 镜像推送请求
type ImagePushRequest struct {
	Image       string `json:"image" binding:"required"`
	RegistryURL string `json:"registryUrl" binding:"required"`
	Namespace   string `json:"namespace"`
}

// composeService 定义 docker-compose service 配置（包内私有）
type composeService struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Environment   []string          `yaml:"environment,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	NetworkMode   string            `yaml:"network_mode,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
	Command       string            `yaml:"command,omitempty"`
	Entrypoint    string            `yaml:"entrypoint,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	User          string            `yaml:"user,omitempty"`
	Hostname      string            `yaml:"hostname,omitempty"`
	Privileged    bool              `yaml:"privileged,omitempty"`
	CapAdd        []string          `yaml:"cap_add,omitempty"`
	CapDrop       []string          `yaml:"cap_drop,omitempty"`
	Deploy        *composeDeploy    `yaml:"deploy,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
}

// composeDeploy 定义资源限制配置
type composeDeploy struct {
	Resources *composeResources `yaml:"resources,omitempty"`
}

// composeResources 定义资源配置
type composeResources struct {
	Limits *composeLimit `yaml:"limits,omitempty"`
}

// composeLimit 定义资源限制
type composeLimit struct {
	Cpus   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// composeFile 定义 docker-compose 文件结构
type composeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]composeService `yaml:"services"`
}
