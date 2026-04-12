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
	Image   string            `json:"image" binding:"required"`
	Name    string            `json:"name"`
	Cmd     []string          `json:"cmd"`
	Env     []string          `json:"env"`
	Ports   map[string]string `json:"ports"`   // e.g., {"8080": "80"}
	Volumes []VolumeMapping   `json:"volumes"` // 目录映射
	Remove  bool              `json:"remove"`
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
