package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/service/metrics"
	"isrvd/server/service/monitor"
)

const metricsContentType = "text/plain; version=0.0.4; charset=utf-8"

// defineMetricsRoutes 定义指标模块路由
func (app *App) defineMetricsRoutes() []Route {
	return []Route{
		// 指标抓取出口（使用 API Token 以 Bearer 方式抓取），各来源独立授权：
		// apisix / caddy 原样透传网关指标，host / containers 导出最后保存的监控记录。
		{Method: "GET", Path: "/metrics/apisix", Handler: app.metricsGatewayScrape(metrics.GatewayApisix), Module: "metrics", Label: "导出 APISIX 指标"},
		{Method: "GET", Path: "/metrics/caddy", Handler: app.metricsGatewayScrape(metrics.GatewayCaddy), Module: "metrics", Label: "导出 Caddy 指标"},
		{Method: "GET", Path: "/metrics/host", Handler: app.metricsLocalExport(metrics.SourceHost), Module: "metrics", Label: "导出本机监控指标"},
		{Method: "GET", Path: "/metrics/containers", Handler: app.metricsLocalExport(metrics.SourceContainers), Module: "metrics", Label: "导出容器监控指标"},
		// 指标状态；本机与容器出口开关在系统配置 metrics 分区，网关指标开关在各网关模块
		{Method: "GET", Path: "/metrics/status", Handler: app.metricsStatus, Module: "metrics", Label: "查询指标状态"},
	}
}

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

// metricsLocalExport 返回导出本机或容器已保存监控记录的处理函数；出口被系统配置禁用时返回 503。
func (app *App) metricsLocalExport(source string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !metrics.LocalEnabled(source) {
			respondError(c, http.StatusServiceUnavailable, "指标出口已禁用")
			return
		}
		if source == metrics.SourceHost {
			data, err := app.monitorCollector.HostMetricsText()
			respondLocalMetrics(c, data, err)
			return
		}
		data, err := app.monitorCollector.ContainersMetricsText(c.Request.Context())
		respondLocalMetrics(c, data, err)
	}
}

func (app *App) metricsStatus(c *gin.Context) {
	respondSuccess(c, "", app.metricsSvc.Status(c.Request.Context()))
}

// ─── 辅助函数 ───

func respondLocalMetrics(c *gin.Context, data []byte, err error) {
	switch {
	case errors.Is(err, monitor.ErrMonitorUnavailable):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	case err != nil:
		respondError(c, http.StatusBadGateway, err.Error())
	default:
		c.Data(http.StatusOK, metricsContentType, data)
	}
}

// respondMetricsToggled 输出网关指标开关结果；conflicts 中的错误映射为 409。
func respondMetricsToggled(c *gin.Context, enabled bool, err error, conflicts ...error) {
	for _, conflict := range conflicts {
		if errors.Is(err, conflict) {
			respondError(c, http.StatusConflict, err.Error())
			return
		}
	}
	switch {
	case err != nil:
		respondError(c, http.StatusBadGateway, err.Error())
	case enabled:
		respondSuccess(c, "指标已开启", nil)
	default:
		respondSuccess(c, "指标已关闭", nil)
	}
}
