package app

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/server/service/system"
)

// running 当前运行的应用实例。StartApp 阻塞到退出，进程内只会有一个。
var running atomic.Pointer[App]

// Auditor 供不经过本进程路由表的入口（中控网关）把操作记入同一份审计日志。
// 审计服务会随配置重载重建，所以这里不持有实例，而是每次取当前运行实例的服务。
type Auditor struct{}

// Audit 返回操作审计入口
func Audit() Auditor { return Auditor{} }

// BodyRead 读取有限长度的请求体用于审计，并保证后续处理仍可读取完整内容。
func (Auditor) BodyRead(r *http.Request) string {
	svc := running.Load().auditService()
	if svc == nil {
		return ""
	}
	return svc.BodyRead(r)
}

// AuditRecord 记录一次操作；请求处理完成后调用。
func (Auditor) AuditRecord(r *http.Request, username, ip string, statusCode int, startTime time.Time, body string) {
	app := running.Load()
	if app == nil {
		logman.Warn("审计服务未就绪，操作未记录", "uri", r.RequestURI)
		return
	}
	// 持读锁完成写入，避免重载在此期间关闭审计服务
	app.servicesMu.RLock()
	defer app.servicesMu.RUnlock()
	if app.auditSvc != nil {
		app.auditSvc.AuditRecord(r, username, ip, statusCode, startTime, body)
	}
}

// auditService 返回当前审计服务；app 为 nil（尚未启动）时返回 nil
func (app *App) auditService() *system.AuditService {
	if app == nil {
		return nil
	}
	app.servicesMu.RLock()
	defer app.servicesMu.RUnlock()
	return app.auditSvc
}
