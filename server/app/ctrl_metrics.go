package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/pkgs/cstore"
	"isrvd/server/service/apisix"
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
		{Method: "GET", Path: "/metrics/host", Handler: app.metricsHost, Module: "metrics", Label: "导出本机监控指标"},
		{Method: "GET", Path: "/metrics/containers", Handler: app.metricsContainers, Module: "metrics", Label: "导出容器监控指标"},
		// 指标状态与开关
		{Method: "GET", Path: "/metrics/status", Handler: app.metricsStatus, Module: "metrics", Label: "查询指标状态"},
		{Method: "PUT", Path: "/metrics/gateway/:name", Handler: app.metricsGatewayUpdate, Module: "metrics", Label: "开关监控指标"},
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

// metricsHost 输出最后保存的主机监控指标。
func (app *App) metricsHost(c *gin.Context) {
	if !metrics.LocalEnabled(metrics.SourceHost) {
		respondError(c, http.StatusServiceUnavailable, "指标出口已禁用")
		return
	}
	data, err := app.monitorCollector.HostMetricsText()
	respondLocalMetrics(c, data, err)
}

// metricsContainers 输出各容器最后保存的资源指标。
func (app *App) metricsContainers(c *gin.Context) {
	if !metrics.LocalEnabled(metrics.SourceContainers) {
		respondError(c, http.StatusServiceUnavailable, "指标出口已禁用")
		return
	}
	data, err := app.monitorCollector.ContainersMetricsText(c.Request.Context())
	respondLocalMetrics(c, data, err)
}

func (app *App) metricsStatus(c *gin.Context) {
	respondSuccess(c, "", app.metricsSvc.Status(c.Request.Context()))
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
