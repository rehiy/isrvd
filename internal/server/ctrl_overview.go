package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/upgrade"

	svcMonitor "isrvd/internal/service/monitor"
)

// defineOverviewRoutes 定义 Overview 模块路由
func (app *App) defineOverviewRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/overview/probe", Handler: app.overviewProbe, Module: "overview", Label: "探测服务可用性", Access: AccessAuth},
		{Method: "GET", Path: "/overview/status", Handler: app.overviewStat, Module: "overview", Label: "获取系统概览统计"},
		{Method: "GET", Path: "/overview/history", Handler: app.overviewMonitorHistory, Module: "overview", Label: "获取监控历史数据"},
		{Method: "POST", Path: "/overview/upgrade", Handler: app.overviewUpgrade, Module: "overview", Label: "升级程序至最新版本"},
	}
}

func (app *App) overviewStat(c *gin.Context) {
	respondSuccess(c, "ok", app.overviewSvc.Stat(c.Request.Context()))
}

func (app *App) overviewProbe(c *gin.Context) {
	respondSuccess(c, "ok", app.overviewSvc.Probe(c.Request.Context(), app.collectProbes()))
}

// collectProbes 收集当前可用服务的探活函数映射
func (app *App) collectProbes() map[string]func(context.Context) bool {
	probes := map[string]func(context.Context) bool{}
	if app.apisixSvc != nil {
		probes["Apisix"] = app.apisixSvc.CheckAvailability
	}
	if app.caddySvc != nil {
		probes["Caddy"] = app.caddySvc.CheckAvailability
	}
	if app.dockerSvc != nil {
		probes["Docker"] = app.dockerSvc.CheckAvailability
	}
	if app.swarmSvc != nil {
		probes["Swarm"] = app.swarmSvc.CheckAvailability
	}
	if app.composeSvc != nil {
		probes["Compose"] = app.composeSvc.CheckAvailability
	}
	return probes
}

func (app *App) overviewUpgrade(c *gin.Context) {
	if err := app.overviewSvc.ApplySelfUpgrade(); err != nil {
		if errors.Is(err, upgrade.ErrNoUpdate) {
			respondError(c, http.StatusBadRequest, "当前已是最新版本")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 先发送响应，延迟重启，确保客户端能收到完整响应
	respondSuccess(c, "升级成功，正在重启", nil)
	go func() {
		time.Sleep(500 * time.Millisecond)
		app.overviewSvc.RestartSelf()
	}()
}

// overviewMonitorHistory 返回监控历史数据
// 查询参数：
//   - type: "host"（默认）或 "container"
//   - id:   容器 ID（type=container 时必填）
//   - since: 时间窗口（秒），默认 3600
func (app *App) overviewMonitorHistory(c *gin.Context) {
	if app.monitorCollector == nil {
		respondError(c, http.StatusServiceUnavailable, "监控采集器未启动")
		return
	}

	sinceStr := c.DefaultQuery("since", "3600")
	since, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil || since <= 0 {
		since = 3600
	}

	switch c.DefaultQuery("type", "host") {
	case "container":
		id := c.Query("id")
		if id == "" {
			respondError(c, http.StatusBadRequest, "缺少容器 ID")
			return
		}
		records, err := svcMonitor.ReadSince[svcMonitor.ContainerRecord](
			app.monitorCollector.DataDir(),
			svcMonitor.ContainerFilePrefix(id),
			since,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, "ok", records)
	default:
		records, err := svcMonitor.ReadSince[svcMonitor.HostRecord](
			app.monitorCollector.DataDir(),
			svcMonitor.HostFilePrefix(),
			since,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, "ok", records)
	}
}
