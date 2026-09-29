package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	svcAccount "isrvd/internal/service/account"
	svcApisix "isrvd/internal/service/apisix"
	svcCaddy "isrvd/internal/service/caddy"
	svcCompose "isrvd/internal/service/compose"
	svcCopilot "isrvd/internal/service/copilot"
	svcCron "isrvd/internal/service/cron"
	svcDocker "isrvd/internal/service/docker"
	svcFiler "isrvd/internal/service/filer"
	svcMonitor "isrvd/internal/service/monitor"
	svcNotify "isrvd/internal/service/notify"
	svcOverview "isrvd/internal/service/overview"
	svcShell "isrvd/internal/service/shell"
	svcSwarm "isrvd/internal/service/swarm"
	svcSystem "isrvd/internal/service/system"
	svcWebSSH "isrvd/internal/service/webssh"

	"isrvd/config"
	"isrvd/internal/registry"
	"isrvd/public"
)

// initServices 初始化/刷新所有业务服务
// 依赖外部服务（apisix/caddy/docker）初始化失败时对应字段为 nil，由 serviceAvailableMiddleware 返回 503
func (app *App) initServices(ctx context.Context) {
	app.overviewSvc = svcOverview.NewService()
	app.configSvc = svcSystem.NewConfigService()
	app.auditSvc = svcSystem.NewAuditService()
	app.accountSvc = svcAccount.NewService()
	app.filerSvc = svcFiler.NewService()
	app.shellSvc = svcShell.NewService()

	app.copilotSvc = svcCopilot.NewService()
	if spec, err := public.Efs.ReadFile("openapi/data.json"); err != nil {
		logman.Warn("copilot catalog unavailable", "error", err)
	} else if err := app.copilotSvc.LoadOpenAPI(spec); err != nil {
		logman.Warn("copilot catalog invalid", "error", err)
	}

	if websshSvc, err := svcWebSSH.NewService(); err != nil {
		logman.Warn("WebSSH service unavailable", "error", err)
		app.websshSvc = nil
	} else {
		app.websshSvc = websshSvc
	}

	// Cron 任务跨服务重载继续执行，仅在整个进程生命周期结束时取消。
	app.cronSvc = svcCron.NewService(app.lifecycleCtx)

	probeCtx, probeCancel := context.WithTimeout(ctx, 5*time.Second)
	apisixSvc, err := svcApisix.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
		app.apisixSvc = nil
	} else {
		app.apisixSvc = apisixSvc
	}

	probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
	caddySvc, err := svcCaddy.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Caddy service unavailable", "error", err)
		app.caddySvc = nil
	} else {
		app.caddySvc = caddySvc
	}

	if dockerSvc, err := svcDocker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
		app.dockerSvc = nil
		app.swarmSvc = nil
	} else {
		app.dockerSvc = dockerSvc
		probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
		swarmSvc, err := svcSwarm.NewService(probeCtx)
		probeCancel()
		if err != nil {
			logman.Warn("Swarm service unavailable", "error", err)
			app.swarmSvc = nil
		} else {
			app.swarmSvc = swarmSvc
		}
	}

	if composeSvc, err := svcCompose.NewService(); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
		app.composeSvc = nil
	} else {
		app.composeSvc = composeSvc
	}

	// 注入可用服务，故障检测独立于监控日志采集。
	sources := svcNotify.FaultSources{
		DockerKey: config.Current().Docker.Host,
		CaddyKey:  config.Current().Caddy.AdminURL,
		ApisixKey: config.Current().Apisix.AdminURL,
	}
	if registry.DockerService != nil {
		sources.Docker = registry.DockerService
	}
	if app.caddySvc != nil {
		sources.Caddy = app.caddySvc
	}
	if app.apisixSvc != nil {
		sources.Apisix = app.apisixSvc
	}
	app.faultWatcher = svcNotify.NewFaultWatcher(config.Current().Notify, sources, app.faultWatcher)
	app.faultWatcher.Start(ctx)

	// 启动后台监控采集
	app.monitorCollector = svcMonitor.NewCollector()
	app.monitorCollector.Start(ctx)
}

// closeServices 释放所有有状态服务持有的资源，并返回旧计划任务全部结束的信号。
func (app *App) closeServices() <-chan struct{} {
	closed := make(chan struct{})
	close(closed)
	var cronDone <-chan struct{} = closed
	if app.cronSvc != nil {
		cronDone = app.cronSvc.Close()
	}
	if app.monitorCollector != nil {
		app.monitorCollector.Stop()
	}
	if app.faultWatcher != nil {
		app.faultWatcher.Stop()
	}
	if app.accountSvc != nil {
		app.accountSvc.Close()
	}
	if app.auditSvc != nil {
		if err := app.auditSvc.Close(); err != nil {
			logman.Warn("close audit service failed", "error", err)
		}
	}
	if app.websshSvc != nil {
		app.websshSvc.Close()
	}
	return cronDone
}

// serviceAvailableMiddleware 根据路由 Module 动态检查服务是否可用，不可用返回 503。
// 匿名/登录即可访问的路由不因可选后端（如未配置 LLM）而拒绝。
func (app *App) serviceAvailableMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		route, ok := lookupRoute(app.routeIndex, c)
		if !ok || route.Access == AccessAnon || route.Access == AccessAuth || app.isServiceAvailable(route.Module) {
			c.Next()
			return
		}
		c.Abort()
		respondError(c, http.StatusServiceUnavailable, route.Label+"服务不可用")
	}
}

// isServiceAvailable 检查指定模块的服务是否可用
func (app *App) isServiceAvailable(module string) bool {
	switch module {
	case "copilot":
		return config.Current().Copilot.BaseURL != ""
	case "apisix":
		return app.apisixSvc != nil
	case "caddy":
		return app.caddySvc != nil
	case "docker":
		return app.dockerSvc != nil
	case "shell":
		return app.shellSvc != nil
	case "swarm":
		return app.dockerSvc != nil && app.swarmSvc != nil
	case "compose":
		return app.dockerSvc != nil && app.composeSvc != nil
	case "ssh":
		return app.websshSvc != nil
	default:
		return true
	}
}
