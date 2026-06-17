package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	pkgDocker "isrvd/pkgs/docker"
)

// NetworkSpec 创建网络请求。
type NetworkSpec struct {
	Name   string `json:"name" binding:"required"` // 网络名称（必填）
	Driver string `json:"driver"`                  // 驱动类型（默认 bridge）
	Subnet string `json:"subnet"`                  // 子网 CIDR（可选）
}

// VolumeSpec 创建卷请求。
type VolumeSpec struct {
	Name   string `json:"name" binding:"required"` // 卷名称（必填）
	Driver string `json:"driver"`                  // 驱动类型（默认 local）
}

// NetworkInfo Docker 网络信息，保持前端稳定响应结构。
type NetworkInfo struct {
	ID     string `json:"id"`     // 网络 ID
	Name   string `json:"name"`   // 网络名称
	Driver string `json:"driver"` // 驱动类型（bridge/overlay/host）
	Subnet string `json:"subnet"` // 子网 CIDR
	Scope  string `json:"scope"`  // 作用域（local/swarm）
}

// NetworkContainerInfo 网络中的容器信息，保持前端稳定响应结构。
type NetworkContainerInfo struct {
	ID         string `json:"id"`         // 容器 ID
	Name       string `json:"name"`       // 容器名称
	IPv4       string `json:"ipv4"`       // 容器 IPv4 地址
	IPv6       string `json:"ipv6"`       // 容器 IPv6 地址
	MacAddress string `json:"macAddress"` // 容器 MAC 地址
}

// NetworkDetail 网络详情响应，保持前端稳定响应结构。
type NetworkDetail struct {
	ID         string                  `json:"id"`         // 网络 ID
	Name       string                  `json:"name"`       // 网络名称
	Driver     string                  `json:"driver"`     // 驱动类型
	Scope      string                  `json:"scope"`      // 作用域（local/swarm）
	Subnet     string                  `json:"subnet"`     // 子网 CIDR
	Gateway    string                  `json:"gateway"`    // 网关地址
	Internal   bool                    `json:"internal"`   // 是否为内部网络
	EnableIPv6 bool                    `json:"enableIPv6"` // 是否启用 IPv6
	Containers []*NetworkContainerInfo `json:"containers"` // 接入该网络的容器列表
}

// VolumeInfo Docker 卷信息，保持前端稳定响应结构。
type VolumeInfo struct {
	Name       string `json:"name"`       // 卷名称
	Driver     string `json:"driver"`     // 驱动类型
	Mountpoint string `json:"mountpoint"` // 挂载点路径
	CreatedAt  string `json:"createdAt"`  // 创建时间
	Size       int64  `json:"size"`       // 卷大小（字节）
}

// VolumeUsedByContainer 使用卷的容器信息，保持前端稳定响应结构。
type VolumeUsedByContainer struct {
	ID        string `json:"id"`        // 容器 ID
	Name      string `json:"name"`      // 容器名称
	MountPath string `json:"mountPath"` // 容器内挂载路径
	ReadOnly  bool   `json:"readOnly"`  // 是否只读挂载
}

// VolumeDetail 数据卷详情响应，保持前端稳定响应结构。
type VolumeDetail struct {
	Name       string                   `json:"name"`       // 卷名称
	Driver     string                   `json:"driver"`     // 驱动类型
	Mountpoint string                   `json:"mountpoint"` // 挂载点路径
	CreatedAt  string                   `json:"createdAt"`  // 创建时间
	Scope      string                   `json:"scope"`      // 作用域（local/global）
	Size       int64                    `json:"size"`       // 卷大小（字节，-1 表示未知）
	RefCount   int64                    `json:"refCount"`   // 引用该卷的容器数（-1 表示未知）
	UsedBy     []*VolumeUsedByContainer `json:"usedBy"`     // 使用该卷的容器列表
}

// NetworkList 列出网络
func (s *Service) NetworkList(ctx context.Context) ([]*NetworkInfo, error) {
	list, err := s.docker.NetworkList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取网络列表失败: %w", err)
	}
	result := make([]*NetworkInfo, 0, len(list))
	for _, net := range list {
		subnet := ""
		if len(net.IPAM.Config) > 0 && net.IPAM.Config[0].Subnet != "" {
			subnet = net.IPAM.Config[0].Subnet
		}
		result = append(result, &NetworkInfo{ID: pkgDocker.ShortID(net.ID), Name: net.Name, Driver: net.Driver, Subnet: subnet, Scope: net.Scope})
	}
	return result, nil
}

// NetworkAction 网络操作
func (s *Service) NetworkAction(ctx context.Context, req ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("网络ID不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.NetworkAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("网络操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// NetworkCreate 创建网络
func (s *Service) NetworkCreate(ctx context.Context, req NetworkSpec) (map[string]string, error) {
	id, err := s.docker.NetworkCreate(ctx, req.Name, req.Driver, req.Subnet)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "name": req.Name}, nil
}

// NetworkInspect 获取网络详情
func (s *Service) NetworkInspect(ctx context.Context, id string) (*NetworkDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("网络ID不能为空")
	}
	netInfo, err := s.docker.NetworkInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取网络详情失败: %w", err)
	}
	return networkDetail(netInfo), nil
}

// VolumeList 列出卷
func (s *Service) VolumeList(ctx context.Context) ([]*VolumeInfo, error) {
	list, err := s.docker.VolumeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取数据卷列表失败: %w", err)
	}
	result := make([]*VolumeInfo, 0, len(list))
	for _, vol := range list {
		if vol == nil {
			continue
		}
		result = append(result, &VolumeInfo{Name: vol.Name, Driver: vol.Driver, Mountpoint: vol.Mountpoint, CreatedAt: vol.CreatedAt})
	}
	return result, nil
}

// VolumeAction 卷操作（ID 字段即卷名）
func (s *Service) VolumeAction(ctx context.Context, req ActionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("数据卷名称不能为空")
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	if err := s.docker.VolumeAction(ctx, req.ID, req.Action); err != nil {
		return fmt.Errorf("数据卷操作 %s 失败: %w", req.Action, err)
	}
	return nil
}

// VolumeCreate 创建卷
func (s *Service) VolumeCreate(ctx context.Context, req VolumeSpec) (map[string]string, error) {
	name, mountpoint, err := s.docker.VolumeCreate(ctx, req.Name, req.Driver)
	if err != nil {
		return nil, err
	}
	return map[string]string{"name": name, "mountpoint": mountpoint}, nil
}

// VolumeInspect 获取卷详情
func (s *Service) VolumeInspect(ctx context.Context, name string) (*VolumeDetail, error) {
	if name == "" {
		return nil, fmt.Errorf("卷名称不能为空")
	}
	volInfo, err := s.docker.VolumeInspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("获取卷详情失败: %w", err)
	}
	containers, _ := s.docker.ContainerList(ctx, true)
	return volumeDetail(volInfo, containers), nil
}

func networkDetail(netInfo network.Inspect) *NetworkDetail {
	containers := make([]*NetworkContainerInfo, 0, len(netInfo.Containers))
	for endpointID, ep := range netInfo.Containers {
		name := ep.Name
		if name == "" {
			name = pkgDocker.ShortID(endpointID)
		}
		containers = append(containers, &NetworkContainerInfo{ID: pkgDocker.ShortID(endpointID), Name: strings.TrimPrefix(name, "/"), IPv4: ep.IPv4Address, IPv6: ep.IPv6Address, MacAddress: ep.MacAddress})
	}
	result := &NetworkDetail{ID: netInfo.ID, Name: netInfo.Name, Driver: netInfo.Driver, Scope: netInfo.Scope, Internal: netInfo.Internal, EnableIPv6: netInfo.EnableIPv6, Containers: containers}
	if len(netInfo.IPAM.Config) > 0 {
		result.Subnet = netInfo.IPAM.Config[0].Subnet
		result.Gateway = netInfo.IPAM.Config[0].Gateway
	}
	return result
}

func volumeDetail(volInfo volume.Volume, containers []container.Summary) *VolumeDetail {
	usedBy := make([]*VolumeUsedByContainer, 0)
	for _, ct := range containers {
		for _, m := range ct.Mounts {
			if m.Type == "volume" && m.Name == volInfo.Name {
				ctName := ""
				if len(ct.Names) > 0 {
					ctName = strings.TrimPrefix(ct.Names[0], "/")
				}
				usedBy = append(usedBy, &VolumeUsedByContainer{ID: pkgDocker.ShortID(ct.ID), Name: ctName, MountPath: m.Destination, ReadOnly: !m.RW})
			}
		}
	}
	// UsageData 仅在带 size 参数查询时返回，可能为 nil；缺省以 -1 表示未知
	size, refCount := int64(-1), int64(-1)
	if volInfo.UsageData != nil {
		size = volInfo.UsageData.Size
		refCount = volInfo.UsageData.RefCount
	}
	return &VolumeDetail{Name: volInfo.Name, Driver: volInfo.Driver, Mountpoint: volInfo.Mountpoint, CreatedAt: volInfo.CreatedAt, Scope: volInfo.Scope, Size: size, RefCount: refCount, UsedBy: usedBy}
}
