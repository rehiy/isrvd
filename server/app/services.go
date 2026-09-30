package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/server/service/account"
	"isrvd/server/service/apisix"
	"isrvd/server/service/caddy"
	"isrvd/server/service/compose"
	"isrvd/server/service/copilot"
	"isrvd/server/service/cron"
	"isrvd/server/service/docker"
	"isrvd/server/service/filer"
	"isrvd/server/service/monitor"
	"isrvd/server/service/notify"
	"isrvd/server/service/overview"
	"isrvd/server/service/shell"
	"isrvd/server/service/swarm"
	"isrvd/server/service/system"
	"isrvd/server/service/webssh"

	pkgDocker "isrvd/pkgs/docker"

	"isrvd/public"
	"isrvd/server/config"
)

// initServices 初始化/刷新所有业务服务
// 依赖外部服务（apisix/caddy/docker）初始化失败时对应字段为 nil，由 serviceAvailableMiddleware 返回 503
func (app *App) initServices(ctx context.Context) {
	app.overviewSvc = overview.NewService()
	app.configSvc = system.NewConfigService()
	app.auditSvc = system.NewAuditService()
	app.accountSvc = account.NewService()
	app.filerSvc = filer.NewService()
	app.shellSvc = shell.NewService()
	app.probes = map[string]func(context.Context) bool{}

	app.copilotSvc = copilot.NewService()
	if spec, err := public.Efs.ReadFile("openapi/data.json"); err != nil {
		logman.Warn("copilot catalog unavailable", "error", err)
	} else if err := app.copilotSvc.LoadOpenAPI(spec); err != nil {
		logman.Warn("copilot catalog invalid", "error", err)
	}

	if websshSvc, err := webssh.NewService(); err != nil {
		logman.Warn("WebSSH service unavailable", "error", err)
		app.websshSvc = nil
	} else {
		app.websshSvc = websshSvc
		app.probes["ssh"] = nil // 本地存储，无需探活
	}

	probeCtx, probeCancel := context.WithTimeout(ctx, 5*time.Second)
	apisixSvc, err := apisix.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
		app.apisixSvc = nil
	} else {
		app.apisixSvc = apisixSvc
		app.probes["apisix"] = apisixSvc.CheckAvailability
	}

	probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
	caddySvc, err := caddy.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Caddy service unavailable", "error", err)
		app.caddySvc = nil
	} else {
		app.caddySvc = caddySvc
		app.probes["caddy"] = caddySvc.CheckAvailability
	}

	var dockerRaw *pkgDocker.DockerService
	if dockerSvc, err := docker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
		app.dockerSvc = nil
		app.swarmSvc = nil
	} else {
		app.dockerSvc = dockerSvc
		app.probes["docker"] = dockerSvc.CheckAvailability
		dockerRaw = dockerSvc.Raw()
		probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
		swarmSvc, err := swarm.NewService(probeCtx, dockerRaw)
		probeCancel()
		if err != nil {
			logman.Warn("Swarm service unavailable", "error", err)
			app.swarmSvc = nil
		} else {
			app.swarmSvc = swarmSvc
			app.probes["swarm"] = swarmSvc.CheckAvailability
		}
	}

	if composeSvc, err := compose.NewService(dockerRaw, app.swarmSvc.Raw()); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
		app.composeSvc = nil
	} else {
		app.composeSvc = composeSvc
		app.probes["compose"] = composeSvc.CheckAvailability
	}

	// Cron 任务跨服务重载继续执行，仅在整个进程生命周期结束时取消。
	app.cronSvc = cron.NewService(app.lifecycleCtx, dockerRaw)

	// 注入可用服务，故障检测独立于监控日志采集。
	sources := notify.FaultSources{
		DockerKey: config.Current().Docker.Host,
		CaddyKey:  config.Current().Caddy.AdminURL,
		ApisixKey: config.Current().Apisix.AdminURL,
	}
	if dockerRaw != nil {
		sources.Docker = dockerRaw
	}
	if app.caddySvc != nil {
		sources.Caddy = app.caddySvc
	}
	if app.apisixSvc != nil {
		sources.Apisix = app.apisixSvc
	}
	app.faultWatcher = notify.NewFaultWatcher(config.Current().Notify, sources, app.faultWatcher)
	app.faultWatcher.Start(ctx)

	// 启动后台监控采集
	app.monitorCollector = monitor.NewCollector(dockerRaw)
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

// optionalModules 依赖外部服务、初始化失败时整体不可用的模块
var optionalModules = map[string]bool{"ssh": true, "apisix": true, "caddy": true, "docker": true, "swarm": true, "compose": true}

// isServiceAvailable 检查指定模块的服务是否可用
func (app *App) isServiceAvailable(module string) bool {
	if module == "copilot" {
		return config.Current().Copilot.BaseURL != ""
	}
	_, ready := app.probes[module]
	return ready || !optionalModules[module]
}
