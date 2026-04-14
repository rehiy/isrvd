package docker

import (
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/server/helper"

)

// ListVolumes 列出卷
func (h *DockerHandler) ListVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	volumes, err := h.dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		logman.Error("List volumes failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取卷列表失败")
		return
	}

	var result []*VolumeInfo
	for _, vol := range volumes.Volumes {
		result = append(result, &VolumeInfo{
			Name: vol.Name, Driver: vol.Driver,
			Mountpoint: vol.Mountpoint, CreatedAt: vol.CreatedAt,
		})
	}
	helper.RespondSuccess(c, "Volumes listed successfully", result)
}

// VolumeAction 卷操作
func (h *DockerHandler) VolumeAction(c *gin.Context) {
	var req VolumeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Volume action failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	switch req.Action {
	case "remove":
		if err := h.dockerClient.VolumeRemove(ctx, req.Name, true); err != nil {
			logman.Error("Remove volume failed", "name", req.Name, "error", err)
			helper.RespondError(c, http.StatusInternalServerError, "删除卷失败: "+err.Error())
			return
		}
	default:
		logman.Error("Unsupported volume action", "action", req.Action)
		helper.RespondError(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}
	logman.Info("Volume action performed", "action", req.Action, "name", req.Name)
	helper.RespondSuccess(c, "Volume action performed successfully", nil)
}

// CreateVolume 创建卷
func (h *DockerHandler) CreateVolume(c *gin.Context) {
	var req VolumeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logman.Error("Create volume failed", "error", err)
		helper.RespondError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	ctx := c.Request.Context()
	driver := req.Driver
	if driver == "" {
		driver = "local"
	}

	resp, err := h.dockerClient.VolumeCreate(ctx, volume.CreateOptions{Name: req.Name, Driver: driver})
	if err != nil {
		logman.Error("Create volume failed", "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "创建卷失败: "+err.Error())
		return
	}
	logman.Info("Volume created", "name", req.Name)
	helper.RespondSuccess(c, "Volume created successfully", gin.H{"name": resp.Name, "mountpoint": resp.Mountpoint})
}

// VolumeInspect 获取卷详情
func (h *DockerHandler) VolumeInspect(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		logman.Error("Volume inspect failed", "error", "volume name is empty")
		helper.RespondError(c, http.StatusBadRequest, "卷名称不能为空")
		return
	}

	ctx := c.Request.Context()

	volInfo, err := h.dockerClient.VolumeInspect(ctx, name)
	if err != nil {
		logman.Error("Volume inspect failed", "name", name, "error", err)
		helper.RespondError(c, http.StatusInternalServerError, "获取卷详情失败: "+err.Error())
		return
	}

	// 查找使用此卷的容器
	var usedBy []*VolumeUsedByContainer
	containers, err := h.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		for _, ct := range containers {
			for _, mount := range ct.Mounts {
				if mount.Type == "volume" && mount.Name == name {
					ctName := ""
					if len(ct.Names) > 0 {
						ctName = strings.TrimPrefix(ct.Names[0], "/")
					}
					usedBy = append(usedBy, &VolumeUsedByContainer{
						ID:        ct.ID[:12],
						Name:      ctName,
						MountPath: mount.Destination,
						ReadOnly:  !mount.RW,
					})
				}
			}
		}
	}

	result := VolumeInspectResponse{
		Name:       volInfo.Name,
		Driver:     volInfo.Driver,
		Mountpoint: volInfo.Mountpoint,
		CreatedAt:  volInfo.CreatedAt,
		Scope:      volInfo.Scope,
		UsedBy:     usedBy,
	}

	if volInfo.UsageData != nil {
		result.Size = volInfo.UsageData.Size
		result.RefCount = volInfo.UsageData.RefCount
	}

	helper.RespondSuccess(c, "Volume details retrieved successfully", result)
}
