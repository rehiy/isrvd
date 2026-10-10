package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/server/i18n"
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
		app.markReady("ssh", nil) // 本地存储，无需探活
	}

	// 中控模式：节点管理由进程内的网关提供（不在本路由表中），这里只向前端声明该能力已启用，
	// 前端据此显示节点切换与节点管理页面
	if app.mode == ModeCenter {
		app.markReady("node", func(context.Context) bool { return true })
	}

	probeCtx, probeCancel := context.WithTimeout(ctx, 5*time.Second)
	apisixSvc, err := apisix.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Apisix service unavailable", "error", err)
		app.apisixSvc = nil
	} else {
		app.apisixSvc = apisixSvc
		app.markReady("apisix", apisixSvc.CheckAvailability)
	}

	probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
	caddySvc, err := caddy.NewService(probeCtx)
	probeCancel()
	if err != nil {
		logman.Warn("Caddy service unavailable", "error", err)
		app.caddySvc = nil
	} else {
		app.caddySvc = caddySvc
		app.markReady("caddy", caddySvc.CheckAvailability)
	}

	var dockerRaw *pkgDocker.DockerService
	if dockerSvc, err := docker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
		app.dockerSvc = nil
		app.swarmSvc = nil
	} else {
		app.dockerSvc = dockerSvc
		app.markReady("docker", dockerSvc.CheckAvailability)
		dockerRaw = dockerSvc.Raw()
		probeCtx, probeCancel = context.WithTimeout(ctx, 5*time.Second)
		swarmSvc, err := swarm.NewService(probeCtx, dockerRaw)
		probeCancel()
		if err != nil {
			logman.Warn("Swarm service unavailable", "error", err)
			app.swarmSvc = nil
		} else {
			app.swarmSvc = swarmSvc
			app.markReady("swarm", swarmSvc.CheckAvailability)
		}
	}

	if composeSvc, err := compose.NewService(dockerRaw, app.swarmSvc.Raw()); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
		app.composeSvc = nil
	} else {
		app.composeSvc = composeSvc
		app.markReady("compose", composeSvc.CheckAvailability)
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
		sources.Caddy = caddyCertSource{app.caddySvc}
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
		// 操作名与提示都按请求语言翻译：先译操作名，再用 %s 套入「服务不可用」
		respondError(c, http.StatusServiceUnavailable,
			fmt.Sprintf(i18n.T(c, "%s服务不可用"), i18n.T(c, route.Label)))
	}
}

// optionalModules 依赖外部服务、初始化失败时整体不可用的模块。
// 新增此类模块时：在此登记，并在 initServices 构造成功后调用 markReady。
var optionalModules = map[string]bool{"ssh": true, "node": true, "apisix": true, "caddy": true, "docker": true, "swarm": true, "compose": true}

// markReady 标记可选模块已就绪并登记探活函数（probe 为 nil 表示无需探活）。
// 登记了未在 optionalModules 声明的模块会被忽略可用性判断，因此这里直接告警。
func (app *App) markReady(module string, probe func(context.Context) bool) {
	if !optionalModules[module] {
		logman.Warn("module not declared in optionalModules", "module", module)
	}
	app.probes[module] = probe
}

// isServiceAvailable 检查指定模块的服务是否可用
func (app *App) isServiceAvailable(module string) bool {
	if module == "copilot" {
		return config.Current().Copilot.BaseURL != ""
	}
	_, ready := app.probes[module]
	return ready || !optionalModules[module]
}

// caddyCertSource 将 Caddy 服务的证书列表适配为告警层所需的精简结构。
type caddyCertSource struct{ svc *caddy.Service }

func (s caddyCertSource) CertList(ctx context.Context) ([]notify.CaddyCert, error) {
	list, err := s.svc.CertList(ctx)
	if err != nil {
		return nil, err
	}
	certs := make([]notify.CaddyCert, 0, len(list))
	for _, c := range list {
		certs = append(certs, notify.CaddyCert{
			Key: c.Key, Source: c.Source, Subject: c.Subject, Certificate: c.Certificate, NotAfter: c.NotAfter,
		})
	}
	return certs, nil
}
