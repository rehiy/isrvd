package docker

import (
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"
	"isrvd/server/model"
)

// ListNetworks 列出网络
func (h *DockerHandler) ListNetworks(c *gin.Context) {
	ctx := c.Request.Context()
	networks, err := h.dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		logman.Error("List networks failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取网络列表失败")
		return
	}

	var result []*model.NetworkInfo
	for _, net := range networks {
		subnet := ""
		if len(net.IPAM.Config) > 0 && net.IPAM.Config[0].Subnet != "" {
			subnet = net.IPAM.Config[0].Subnet
		}
		id := net.ID
		if len(id) > 12 {
			id = id[:12]
		}
		result = append(result, &model.NetworkInfo{
			ID: id, Name: net.Name, Driver: net.Driver, Subnet: subnet, Scope: net.Scope,
		})
	}
	helper.RespondSuccess(c, "Networks listed successfully", result)
}

// NetworkAction 网络操作
func (h *DockerHandler) NetworkAction(c *gin.Context) {
	var req model.NetworkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Network action failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	switch req.Action {
	case "remove":
		if err := h.dockerClient.NetworkRemove(ctx, req.ID); err != nil {
			logman.Error("Remove network failed", "id", req.ID, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "删除网络失败: "+err.Error())
			return
		}
	default:
		logman.Error("Unsupported network action", "action", req.Action)
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Network action performed", "action", req.Action, "id", req.ID)
	helper.RespondSuccess(c, "Network action performed successfully", nil)
}

// CreateNetwork 创建网络
func (h *DockerHandler) CreateNetwork(c *gin.Context) {
	var req model.NetworkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create network failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	driver := req.Driver
	if driver == "" {
		driver = "bridge"
	}

	resp, err := h.dockerClient.NetworkCreate(ctx, req.Name, types.NetworkCreate{Driver: driver})
	if err != nil {
		logman.Error("Create network failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建网络失败: "+err.Error())
		return
	}
	id := resp.ID
	if len(id) > 12 {
		id = id[:12]
	}
	logman.Info("Network created", "name", req.Name, "id", id)
	helper.RespondSuccess(c, "Network created successfully", gin.H{"id": id, "name": req.Name})
}

// NetworkInspect 获取网络详情
func (h *DockerHandler) NetworkInspect(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		logman.Error("Network inspect failed", "error", "empty network ID")
		helper.RespondError(c, http.StatusBadRequest, "网络 ID 不能为空")
		return
	}

	ctx := c.Request.Context()

	networkInfo, err := h.dockerClient.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		logman.Error("Network inspect failed", "id", id, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取网络详情失败: "+err.Error())
		return
	}

	// 提取已连接的容器信息
	var containers []*model.NetworkContainerInfo
	for endpointID, ep := range networkInfo.Containers {
		name := ep.Name
		if name == "" {
			name = endpointID[:12]
		}
		// 尝试获取容器名称
		containerJSON, err := h.dockerClient.ContainerInspect(ctx, ep.Name)
		if err == nil && len(containerJSON.Name) > 0 {
			name = strings.TrimPrefix(containerJSON.Name, "/")
		}
		containers = append(containers, &model.NetworkContainerInfo{
			ID:         endpointID[:12],
			Name:       name,
			IPv4:       ep.IPv4Address,
			IPv6:       ep.IPv6Address,
			MacAddress: ep.MacAddress,
		})
	}

	result := model.NetworkInspectResponse{
		ID:         networkInfo.ID,
		Name:       networkInfo.Name,
		Driver:     networkInfo.Driver,
		Scope:      networkInfo.Scope,
		Internal:   networkInfo.Internal,
		EnableIPv6: networkInfo.EnableIPv6,
		Containers: containers,
	}

	if len(networkInfo.IPAM.Config) > 0 {
		result.Subnet = networkInfo.IPAM.Config[0].Subnet
		result.Gateway = networkInfo.IPAM.Config[0].Gateway
	}

	helper.RespondSuccess(c, "Network details retrieved successfully", result)
}
