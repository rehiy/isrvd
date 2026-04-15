package system

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/psutil"

	"isrvd/server/helper"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// SystemStatResponse 系统统计响应
type SystemStatResponse struct {
	System *psutil.DetailStat `json:"system"`
	Go     *GoRuntimeStat     `json:"go"`
}

// GoRuntimeStat Go 运行态统计
type GoRuntimeStat struct {
	Version      string `json:"version"`
	NumCPU       int    `json:"numCPU"`
	NumGoroutine int    `json:"numGoroutine"`
	*psutil.GoMemoryStat
}

// Stat 获取系统统计信息
func (h *SystemHandler) Stat(c *gin.Context) {
	detail := psutil.Detail(false)

	goStat := &GoRuntimeStat{
		Version:      runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		GoMemoryStat: psutil.GoMemory(),
	}

	helper.RespondSuccess(c, "ok", &SystemStatResponse{
		System: detail,
		Go:     goStat,
	})
}

// UpTime 获取服务启动时间（秒级时间戳）
var startTime = time.Now()

func (h *SystemHandler) Uptime(c *gin.Context) {
	helper.RespondSuccess(c, "ok", gin.H{
		"startTime": startTime.Unix(),
		"uptime":    int64(time.Since(startTime).Seconds()),
	})
}

// Health 健康检查
func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
