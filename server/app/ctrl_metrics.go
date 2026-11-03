package app

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"isrvd/pkgs/cstore"
	"isrvd/server/i18n"
	"isrvd/server/service/apisix"
	"isrvd/server/service/metrics"
	"isrvd/server/service/monitor"
)

// defineMetricsRoutes 定义指标模块路由
func (app *App) defineMetricsRoutes() []Route {
	return []Route{
		// 指标抓取出口（使用 API Token 以 Bearer 方式抓取），各来源独立授权：
		// apisix / caddy 原样透传网关指标，host 为 isrvd 所在主机的监控指标，containers 为运行中容器的资源指标。
		{Method: "GET", Path: "/metrics/apisix", Handler: app.metricsGatewayScrape(metrics.GatewayApisix), Module: "metrics", Label: "导出 APISIX 指标"},
		{Method: "GET", Path: "/metrics/caddy", Handler: app.metricsGatewayScrape(metrics.GatewayCaddy), Module: "metrics", Label: "导出 Caddy 指标"},
		{Method: "GET", Path: "/metrics/host", Handler: app.metricsHost, Module: "metrics", Label: "导出本机监控指标"},
		{Method: "GET", Path: "/metrics/containers", Handler: app.metricsContainers, Module: "metrics", Label: "导出容器监控指标"},
		// 内置统计页
		{Method: "GET", Path: "/metrics/status", Handler: app.metricsStatus, Module: "metrics", Label: "查询指标状态"},
		{Method: "GET", Path: "/metrics/stats", Handler: app.metricsStats, Module: "metrics", Label: "查询网关访问统计"},
		{Method: "PUT", Path: "/metrics/gateway/:name", Handler: app.metricsGatewayUpdate, Module: "metrics", Label: "开关监控指标"},
	}
}

const metricsContentType = "text/plain; version=0.0.4; charset=utf-8"

// metricsGatewayScrape 返回透传单个网关指标的处理函数。
// 网关未配置返回 404，抓取失败返回 502（抓取端据此判定目标不可用）。
func (app *App) metricsGatewayScrape(gateway string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := app.metricsSvc.Scrape(c.Request.Context(), gateway)
		switch {
		case errors.Is(err, metrics.ErrGatewayUnknown):
			respondError(c, http.StatusNotFound, "网关未配置")
		case err != nil:
			respondError(c, http.StatusBadGateway, err.Error())
		default:
			c.Data(http.StatusOK, metricsContentType, data)
		}
	}
}

// metricsHost 以指标文本格式输出 isrvd 所在主机的实时监控指标
func (app *App) metricsHost(c *gin.Context) {
	if !metrics.LocalEnabled(metrics.SourceHost) {
		respondError(c, http.StatusServiceUnavailable, "指标出口已禁用")
		return
	}
	c.Data(http.StatusOK, metricsContentType, monitor.CollectHostStat(c.Request.Context()).MetricsText())
}

// metricsContainers 以指标文本格式输出运行中容器的实时资源指标。
// 每个容器采样约需 1 秒（并发采集），Docker 不可用或出口禁用返回 503。
func (app *App) metricsContainers(c *gin.Context) {
	if !metrics.LocalEnabled(metrics.SourceContainers) {
		respondError(c, http.StatusServiceUnavailable, "指标出口已禁用")
		return
	}
	data, err := app.monitorCollector.ContainersMetricsText(c.Request.Context())
	switch {
	case errors.Is(err, monitor.ErrDockerUnavailable):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	case err != nil:
		respondError(c, http.StatusBadGateway, err.Error())
	default:
		c.Data(http.StatusOK, metricsContentType, data)
	}
}

func (app *App) metricsStatus(c *gin.Context) {
	respondSuccess(c, "", app.metricsSvc.Status(c.Request.Context()))
}

// metricsStats 查询参数 since：统计窗口（秒），默认 3600，服务端限制在 60 秒到 24 小时之间
func (app *App) metricsStats(c *gin.Context) {
	since, err := strconv.ParseInt(c.DefaultQuery("since", "3600"), 10, 64)
	if err != nil {
		since = 3600
	}
	stats := app.metricsSvc.Stats(since)
	for i := range stats {
		stats[i].Error = i18n.TC(c.Request.Context(), stats[i].Error)
	}
	respondSuccess(c, "", stats)
}

func (app *App) metricsGatewayUpdate(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled" binding:"required"` // true 开启，false 关闭
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	name := c.Param("name")
	if err := app.metricsSvc.Update(c.Request.Context(), name, *req.Enabled); err != nil {
		switch {
		case errors.Is(err, metrics.ErrGatewayUnknown):
			respondError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, metrics.ErrGatewayUnavailable):
			respondError(c, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, apisix.ErrMetricsExternallyManaged), errors.Is(err, cstore.ErrConflict):
			respondError(c, http.StatusConflict, err.Error())
		case name == metrics.SourceHost || name == metrics.SourceContainers:
			respondError(c, http.StatusInternalServerError, err.Error())
		default:
			respondError(c, http.StatusBadGateway, err.Error())
		}
		return
	}
	if *req.Enabled {
		respondSuccess(c, "指标已开启", nil)
		return
	}
	respondSuccess(c, "指标已关闭", nil)
}
