package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/upgrade"

	"isrvd/config"
	svcAccount "isrvd/internal/service/account"
	svcMonitor "isrvd/internal/service/monitor"
	svcOverview "isrvd/internal/service/overview"
)

// defineOverviewRoutes 定义 Overview 模块路由
func (app *App) defineOverviewRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/overview/bootstrap", Handler: app.overviewBootstrap, Module: "overview", Label: "获取启动数据", Access: AccessAnon},
		{Method: "GET", Path: "/overview/monitor", Handler: app.overviewMonitor, Module: "overview", Label: "获取监控数据"},
		{Method: "GET", Path: "/overview/version", Handler: app.overviewVersion, Module: "overview", Label: "获取版本信息"},
		{Method: "POST", Path: "/overview/upgrade", Handler: app.overviewUpgrade, Module: "overview", Label: "升级程序至最新版本"},
	}
}

// BootstrapConfig 启动所需的最小系统配置
type BootstrapConfig struct {
	MaxUploadSize  int64                `json:"maxUploadSize"`
	MarketplaceURL string               `json:"marketplaceUrl"`
	OpenAPIEnabled bool                 `json:"openapiEnabled"`
	Links          []*config.LinkConfig `json:"links"`
}

// BootstrapResponse 前端启动所需的聚合数据
type BootstrapResponse struct {
	Auth   *svcAccount.AuthInfoResponse `json:"auth"`             // 认证信息
	Probe  *svcOverview.ProbeResponse   `json:"probe,omitempty"`  // 服务探活结果
	Config *BootstrapConfig             `json:"config,omitempty"` // 系统配置
}

// overviewBootstrap 聚合启动所需数据：auth + probe + config
// AccessAnon：未登录也可调用，probe/config 仅登录后返回
func (app *App) overviewBootstrap(c *gin.Context) {
	ctx := c.Request.Context()
	username := c.GetString("username")

	resp := &BootstrapResponse{
		Auth: app.accountSvc.AuthInfo(username),
	}

	// 已登录时并发获取 probe + config
	if username != "" {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			resp.Probe = app.overviewSvc.Probe(ctx, app.collectProbes())
		}()
		go func() {
			defer wg.Done()
			resp.Config = &BootstrapConfig{
				MaxUploadSize:  config.Server.MaxUploadSize,
				MarketplaceURL: config.Marketplace.URL,
				OpenAPIEnabled: config.Server.OpenAPI,
				Links:          config.Links,
			}
		}()
		wg.Wait()
	}

	respondSuccess(c, "ok", resp)
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

func (app *App) overviewVersion(c *gin.Context) {
	respondSuccess(c, "ok", app.overviewSvc.CheckVersion())
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

// overviewMonitor 返回监控数据
// 查询参数：
//   - type:  "host"（默认）或 "container"
//   - id:    容器 ID（type=container 时必填）
//   - since: 时间窗口（秒），默认 3600；传 0 为实时模式（直接采集当前数据，不写入文件）
func (app *App) overviewMonitor(c *gin.Context) {
	if app.monitorCollector == nil {
		respondError(c, http.StatusServiceUnavailable, "监控采集器未启动")
		return
	}

	sinceStr := c.DefaultQuery("since", "3600")
	since, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		since = 3600
	}

	// since=0：实时模式
	if since == 0 {
		app.overviewMonitorRealtime(c)
		return
	}

	if since < 0 {
		since = 3600
	}

	switch c.DefaultQuery("type", "host") {
	case "container":
		id := c.Query("id")
		if id == "" {
			respondError(c, http.StatusBadRequest, "缺少容器 ID")
			return
		}
		records, err := svcMonitor.ReadSince[svcMonitor.Record](
			app.monitorCollector.DataDir(),
			svcMonitor.ContainerPrefix,
			id,
			since,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, "ok", records)
	default:
		records, err := svcMonitor.ReadSince[svcMonitor.Record](
			app.monitorCollector.DataDir(),
			svcMonitor.HostPrefix,
			"", // containerID 为空表示查询主机
			since,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, "ok", records)
	}
}

// overviewMonitorRealtime 实时模式：直接采集当前数据，不写入文件
func (app *App) overviewMonitorRealtime(c *gin.Context) {
	ctx := c.Request.Context()

	switch c.DefaultQuery("type", "host") {
	case "container":
		id := c.Query("id")
		if id == "" {
			respondError(c, http.StatusBadRequest, "缺少容器 ID")
			return
		}
		respondSuccess(c, "ok", app.monitorCollector.CollectContainerStatNow(ctx, id))
	default:
		respondSuccess(c, "ok", app.monitorCollector.CollectHostStatNow(ctx))
	}
}
