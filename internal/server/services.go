package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
func (app *App) initServices() {
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

	app.cronSvc = svcCron.NewService()

	if apisixSvc, err := svcApisix.NewService(); err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
		app.apisixSvc = nil
	} else {
		app.apisixSvc = apisixSvc
	}

	if caddySvc, err := svcCaddy.NewService(); err != nil {
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
		if swarmSvc, err := svcSwarm.NewService(); err != nil {
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
		DockerKey: config.Docker.Host,
		CaddyKey:  config.Caddy.AdminURL,
		ApisixKey: config.Apisix.AdminURL,
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
	app.faultWatcher = svcNotify.NewFaultWatcher(config.Notify, sources, app.faultWatcher)
	app.faultWatcher.Start(context.Background())

	// 启动后台监控采集
	app.monitorCollector = svcMonitor.NewCollector()
	app.monitorCollector.Start(context.Background())
}

// closeServices 释放所有有状态服务持有的资源
func (app *App) closeServices() {
	if app.cronSvc != nil {
		done := app.cronSvc.Close()
		app.cronWG.Go(func() { <-done })
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
		return config.Copilot.BaseURL != ""
	case "apisix":
		return app.apisixSvc != nil
	case "caddy":
		return app.caddySvc != nil
	case "docker", "shell":
		return app.dockerSvc != nil
	case "swarm":
		return app.dockerSvc != nil && app.swarmSvc != nil
	case "compose":
		return app.dockerSvc != nil && app.composeSvc != nil
	case "webssh":
		return app.websshSvc != nil
	default:
		return true
	}
}

// watchReload 统一持有 HTTP 服务与信号生命周期，避免多个退出回调抢先终止进程。
func (app *App) watchReload(server *http.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	// 单个协程串行重载，退出时等待它结束，避免与服务清理并发。
	reloadDone := make(chan struct{})
	go func() {
		defer close(reloadDone)
		for {
			select {
			case <-ctx.Done():
				return
			case <-hup:
				logman.Info("received SIGHUP, reloading...")
			case <-config.ReloadCh:
				logman.Info("config changed, reloading...")
			}
			if ctx.Err() != nil {
				return
			}
			app.reload()
		}
	}()
	listenErr := make(chan error, 1)
	go func() {
		logman.Info("httpd start", "address", server.Addr)
		listenErr <- server.ListenAndServe()
	}()
	select {
	case err := <-listenErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logman.Error("httpd server stopped", "error", err)
		}
	case <-ctx.Done():
		logman.Info("received signal, shutting down...")
	}
	stop()
	app.shutdown(server, reloadDone)
}

// shutdown 让 HTTP 请求、所有新旧任务及通知发送共用退出宽限期。
func (app *App) shutdown(server *http.Server, reloadDone <-chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() {
		if !svcNotify.Shutdown(ctx) {
			logman.Warn("notification shutdown grace period expired")
		}
	}()
	if err := server.Shutdown(ctx); err != nil {
		logman.Warn("httpd shutdown grace period expired", "error", err)
		_ = server.Close()
	}
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		<-reloadDone
		if ctx.Err() != nil {
			return
		}
		app.closeServices()
		app.cronWG.Wait()
	}()
	select {
	case <-closed:
	case <-ctx.Done():
		logman.Warn("service shutdown grace period expired")
	}
}

// reload 重新加载配置和服务
func (app *App) reload() {
	if err := config.Load(); err != nil {
		logman.Error("config reload failed", "error", err)
		return
	}
	// 关闭旧服务持有的资源，再重新初始化（含监控采集器）
	app.closeServices()
	registry.Init()
	app.initServices()
	logman.Info("reload complete")
}
