package app

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
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
	"isrvd/server/service/metrics"
	"isrvd/server/service/monitor"
	"isrvd/server/service/notify"
	"isrvd/server/service/overview"
	"isrvd/server/service/shell"
	"isrvd/server/service/swarm"
	"isrvd/server/service/system"
	"isrvd/server/service/webssh"

	"isrvd/public"
	"isrvd/server/config"
)

// initServices 初始化/刷新所有业务服务
// 依赖外部服务（apisix/caddy/docker）初始化失败时对应字段为 nil，由 serviceAvailableMiddleware 返回 503
func (app *App) initServices(ctx context.Context) {
	available := &availableServices{probes: map[string]func(context.Context) bool{}}
	app.overviewSvc = overview.NewService()
	app.configSvc = system.NewConfigService()
	app.auditSvc = system.NewAuditService()
	app.accountSvc = account.NewService()
	app.filerSvc = filer.NewService()
	app.shellSvc = shell.NewService()

	app.copilotSvc = copilot.NewService()
	if spec, err := public.Efs.ReadFile("openapi/data.json"); err != nil {
		logman.Warn("copilot catalog unavailable", "error", err)
	} else if err := app.copilotSvc.LoadOpenAPI(spec); err != nil {
		logman.Warn("copilot catalog invalid", "error", err)
	}

	var err error
	if app.websshSvc, err = webssh.NewService(); err != nil {
		logman.Warn("WebSSH service unavailable", "error", err)
	} else {
		available.markReady("ssh", nil) // 本地存储，无需探活
	}

	// 中控模式：节点管理由进程内的网关提供（不在本路由表中），这里只向前端声明该能力已启用，
	// 前端据此显示节点切换与节点管理页面
	if config.IsCenter() {
		available.markReady("node", func(context.Context) bool { return true })
	}

	if app.dockerSvc, err = docker.NewService(); err != nil {
		logman.Warn("Docker service unavailable", "error", err)
	} else {
		available.markReady("docker", app.dockerSvc.CheckAvailability)
	}
	app.initProbedServices(ctx, available)

	// Docker 不可用时 app.dockerSvc 为 nil，Raw() 对 nil 接收者安全并返回 nil
	dockerRaw := app.dockerSvc.Raw()
	if app.composeSvc, err = compose.NewService(dockerRaw, app.swarmSvc.Raw()); err != nil {
		logman.Warn("Compose service unavailable", "error", err)
	} else {
		available.markReady("compose", app.composeSvc.CheckAvailability)
	}

	// Cron 任务跨服务重载继续执行，仅在整个进程生命周期结束时取消。
	app.cronSvc = cron.NewService(app.lifecycleCtx, dockerRaw)

	// 启动后台监控采集
	app.monitorCollector = monitor.NewCollector(dockerRaw)
	app.monitorCollector.Start(ctx)

	app.publishAvailableServices(ctx, available)
}

// availableServices 发布后不可修改。配置重载与网关重试由同一循环串行写入。
type availableServices struct {
	apisixSvc  *apisix.Service
	caddySvc   *caddy.Service
	metricsSvc *metrics.Service
	probes     map[string]func(context.Context) bool
}

func (app *App) publishAvailableServices(ctx context.Context, available *availableServices) {
	dockerRaw := app.dockerSvc.Raw()
	cfg := config.Current()
	// 指标统一读出口；只登记已配置抓取地址的网关，管理服务不可用时不提供开关状态
	available.metricsSvc = metrics.NewService(dockerRaw != nil)
	if u := apisix.MetricsURL(cfg.Apisix); u != "" {
		var status func(context.Context) (bool, error)
		if available.apisixSvc != nil {
			status = available.apisixSvc.MetricsStatus
		}
		available.metricsSvc.AddGateway(metrics.GatewayApisix, u, status)
	}
	if u := caddy.MetricsURL(cfg.Caddy); u != "" {
		var status func(context.Context) (bool, error)
		if available.caddySvc != nil {
			status = available.caddySvc.MetricsStatus
		}
		available.metricsSvc.AddGateway(metrics.GatewayCaddy, u, status)
	}
	app.availability.Store(available)

	// 注入可用服务，故障检测独立于监控日志采集。
	sources := notify.FaultSources{
		DockerKey: cfg.Docker.Host,
		CaddyKey:  cfg.Caddy.AdminURL,
		ApisixKey: cfg.Apisix.AdminURL,
	}
	if dockerRaw != nil {
		sources.Docker = dockerRaw
	}
	if available.caddySvc != nil {
		sources.Caddy = caddyCertSource{available.caddySvc}
	}
	if available.apisixSvc != nil {
		sources.Apisix = available.apisixSvc
	}
	if app.faultWatcher != nil {
		app.faultWatcher.Stop()
	}
	app.faultWatcher = notify.NewFaultWatcher(cfg.Notify, sources, app.faultWatcher)
	app.faultWatcher.Start(ctx)
}

// initProbedServices 并行初始化需要访问外部服务的模块，探活共用 5 秒预算。
func (app *App) initProbedServices(ctx context.Context, available *availableServices) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var apisixErr, caddyErr, swarmErr error
	wg.Go(func() { available.apisixSvc, apisixErr = apisix.NewService(ctx) })
	wg.Go(func() { available.caddySvc, caddyErr = caddy.NewService(ctx) })
	app.swarmSvc = nil
	if dockerRaw := app.dockerSvc.Raw(); dockerRaw != nil {
		wg.Go(func() { app.swarmSvc, swarmErr = swarm.NewService(ctx, dockerRaw) })
	}
	wg.Wait()

	if apisixErr != nil {
		logman.Warn("Apisix service unavailable", "error", apisixErr)
	} else {
		available.markReady("apisix", available.apisixSvc.CheckAvailability)
	}
	if caddyErr != nil {
		logman.Warn("Caddy service unavailable", "error", caddyErr)
	} else {
		available.markReady("caddy", available.caddySvc.CheckAvailability)
	}
	if swarmErr != nil {
		logman.Warn("Swarm service unavailable", "error", swarmErr)
	} else if app.swarmSvc != nil {
		available.markReady("swarm", app.swarmSvc.CheckAvailability)
	}
}

// retryGateways 只重试已配置、初始化失败的网关。
func (app *App) retryGateways(ctx context.Context) {
	current := app.availability.Load()
	cfg := config.Current()
	wantApisix := current.apisixSvc == nil && cfg.Apisix.AdminURL != ""
	wantCaddy := current.caddySvc == nil && cfg.Caddy.AdminURL != ""
	if !wantApisix && !wantCaddy {
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var apisixSvc *apisix.Service
	var caddySvc *caddy.Service
	var wg sync.WaitGroup
	if wantApisix {
		wg.Go(func() { apisixSvc, _ = apisix.NewService(probeCtx) })
	}
	if wantCaddy {
		wg.Go(func() { caddySvc, _ = caddy.NewService(probeCtx) })
	}
	wg.Wait()
	if ctx.Err() != nil || apisixSvc == nil && caddySvc == nil {
		return
	}
	available := *current
	available.probes = maps.Clone(current.probes)
	if apisixSvc != nil {
		available.apisixSvc = apisixSvc
		available.markReady("apisix", apisixSvc.CheckAvailability)
	}
	if caddySvc != nil {
		available.caddySvc = caddySvc
		available.markReady("caddy", caddySvc.CheckAvailability)
	}
	app.publishAvailableServices(app.servicesCtx, &available)
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
func (available *availableServices) markReady(module string, probe func(context.Context) bool) {
	if !optionalModules[module] {
		logman.Warn("module not declared in optionalModules", "module", module)
	}
	available.probes[module] = probe
}

// isServiceAvailable 检查指定模块的服务是否可用
func (app *App) isServiceAvailable(module string) bool {
	if module == "copilot" {
		return config.Current().Copilot.BaseURL != ""
	}
	if !optionalModules[module] {
		return true
	}
	_, ready := app.availability.Load().probes[module]
	return ready
}

// caddyCertSource 将 Caddy 服务的证书列表适配为告警层所需的精简结构。
type caddyCertSource struct{ svc *caddy.Service }

func (s caddyCertSource) CertList(ctx context.Context) ([]notify.CaddyCert, bool, error) {
	list, complete, err := s.svc.CertListForAlerts(ctx)
	if err != nil {
		return nil, false, err
	}
	certs := make([]notify.CaddyCert, 0, len(list))
	for _, c := range list {
		certs = append(certs, notify.CaddyCert{
			Key: c.Key, AlertKey: c.AlertKey, Subject: c.Subject, NotAfter: c.NotAfter,
		})
	}
	return certs, complete, nil
}
