package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/server/config"
	"isrvd/server/service/notify"
)

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

func (app *App) cancelServiceRequests() {
	app.requestCtxMu.RLock()
	cancel := app.requestsCancel
	app.requestCtxMu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func (app *App) resetRequestContext() {
	ctx, cancel := context.WithCancel(app.lifecycleCtx)
	app.requestCtxMu.Lock()
	app.requestsCtx = ctx
	app.requestsCancel = cancel
	app.requestCtxMu.Unlock()
}

// shutdown 先等待 HTTP 请求，再为后台任务提供独立宽限期。
func (app *App) shutdown(server *http.Server, reloadDone <-chan struct{}) {
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(httpCtx); err != nil {
		logman.Warn("httpd shutdown grace period expired", "error", err)
		_ = server.Close()
	}
	httpCancel()

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		<-reloadDone
		app.servicesMu.Lock()
		cronDone := app.closeServices()
		dockerSvc := app.dockerSvc
		app.servicesMu.Unlock()
		app.cleanupWG.Go(func() { <-cronDone })
		app.cleanupWG.Wait()
		if dockerRaw := dockerSvc.Raw(); dockerRaw != nil {
			if err := dockerRaw.Close(); err != nil {
				logman.Warn("Docker service close failed", "error", err)
			}
		}
	}()

	serviceTimer := time.NewTimer(5 * time.Second)
	select {
	case <-closed:
		serviceTimer.Stop()
	case <-serviceTimer.C:
		logman.Warn("service shutdown grace period expired")
		app.lifecycleCancel()
		app.cancelServiceRequests()
		cleanupTimer := time.NewTimer(time.Second)
		select {
		case <-closed:
			cleanupTimer.Stop()
		case <-cleanupTimer.C:
		}
	}

	app.lifecycleCancel()
	app.cancelServiceRequests()
	notifyCtx, notifyCancel := context.WithTimeout(context.Background(), time.Second)
	defer notifyCancel()
	if !notify.Shutdown(notifyCtx) {
		logman.Warn("notification shutdown grace period expired")
	}
}

// reload 等待当前请求结束后读取并发布同一份配置候选，再重建服务。
func (app *App) reload() {
	app.cancelServiceRequests()
	app.servicesMu.Lock()
	defer app.servicesMu.Unlock()

	candidate, err := config.ReadStored()
	if err != nil {
		logman.Error("config reload rejected, keeping previous runtime", "error", err)
		app.resetRequestContext()
		return
	}
	if candidate == nil {
		app.resetRequestContext()
		return
	}
	config.Publish(candidate)
	app.wsConfig.AllowedOrigins = append([]string(nil), candidate.Server.AllowedOrigins...)

	oldDockerRaw := app.dockerSvc.Raw()
	if app.servicesCancel != nil {
		app.servicesCancel()
	}
	cronDone := app.closeServices()
	app.cleanupWG.Go(func() {
		<-cronDone
		if oldDockerRaw != nil {
			if err := oldDockerRaw.Close(); err != nil {
				logman.Warn("old Docker service close failed", "error", err)
			}
		}
	})

	app.servicesCtx, app.servicesCancel = context.WithCancel(app.lifecycleCtx)
	app.initServices(app.servicesCtx)
	app.resetRequestContext()
	logman.Info("reload complete")
}
