package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	svcAccount "isrvd/internal/service/account"
	svcAgent "isrvd/internal/service/agent"
	svcApisix "isrvd/internal/service/apisix"
	svcCaddy "isrvd/internal/service/caddy"
	svcCompose "isrvd/internal/service/compose"
	svcCron "isrvd/internal/service/cron"
	svcDocker "isrvd/internal/service/docker"
	svcFiler "isrvd/internal/service/filer"
	svcOverview "isrvd/internal/service/overview"
	svcShell "isrvd/internal/service/shell"
	svcSwarm "isrvd/internal/service/swarm"
	svcSystem "isrvd/internal/service/system"
	svcWebSSH "isrvd/internal/service/webssh"

	"isrvd/config"
	"isrvd/internal/registry"
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
	app.agentSvc = svcAgent.NewService()

	if websshSvc, err := svcWebSSH.NewService(); err != nil {
		logman.Warn("WebSSH service unavailable", "error", err)
		app.websshSvc = nil
	} else {
		app.websshSvc = websshSvc
	}

	app.cronSvc = svcCron.NewService(registry.DockerService)

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
}

// serviceAvailableMiddleware 根据路由 Module 动态检查服务是否可用，不可用返回 503
func (app *App) serviceAvailableMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		route, ok := app.routeIndex[c.Request.Method+" "+c.FullPath()]
		if !ok || app.isServiceAvailable(route.Module) {
			c.Next()
			return
		}
		c.Abort()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  fmt.Sprintf("%s service unavailable", route.Module),
			"module": route.Module,
			"label":  route.Label,
			"reload": "send SIGHUP to reload services",
		})
	}
}

// isServiceAvailable 检查指定模块的服务是否可用
func (app *App) isServiceAvailable(module string) bool {
	switch module {
	case "agent":
		return config.Agent.BaseURL != ""
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

// watchReload 监听 SIGHUP 信号和 etcd 配置变更，触发重载
func (app *App) watchReload() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-sig:
				logman.Info("received SIGHUP, reloading...")
				app.reload()
			case <-config.ReloadCh:
				logman.Info("config changed, reloading...")
				app.reload()
			}
		}
	}()
}

// reload 重新加载配置和服务
func (app *App) reload() {
	if err := config.Load(); err != nil {
		logman.Error("config reload failed", "error", err)
		return
	}
	registry.Init()
	app.initServices()
	logman.Info("reload complete")
}
