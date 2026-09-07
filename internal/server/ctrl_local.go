package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	svcLocal "isrvd/internal/service/local"
)

// defineLocalRoutes 定义本机管理模块路由
func (app *App) defineLocalRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/local/processes", Handler: app.localProcessList, Module: "local", Label: "查询进程列表"},
		// 终止进程是高危写操作，强制记审计
		{Method: "POST", Path: "/local/process/:pid/kill", Handler: app.localProcessKill, Module: "local", Label: "终止进程", Audit: AuditAlways},
	}
}

// ─── Handler 方法 ───

func (app *App) localProcessList(c *gin.Context) {
	includeCmdline := app.accountSvc.FounderCheck(c.GetString("username")) == nil
	list, err := svcLocal.ProcessList(includeCmdline)
	if err != nil {
		logman.Error("获取进程列表失败", "error", err)
		respondError(c, http.StatusInternalServerError, "获取进程列表失败")
		return
	}
	respondSuccess(c, "获取进程列表成功", gin.H{"processes": list})
}

func (app *App) localProcessKill(c *gin.Context) {
	if err := app.accountSvc.FounderCheck(c.GetString("username")); err != nil {
		respondError(c, http.StatusForbidden, err.Error())
		return
	}

	pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
	if err != nil || pid <= 0 {
		respondError(c, http.StatusBadRequest, "进程 ID 无效")
		return
	}

	// 无请求体时按优雅终止处理
	var req struct {
		Force bool `json:"force"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := svcLocal.ProcessKill(int32(pid), req.Force); err != nil {
		logman.Warn("终止进程失败", "pid", pid, "error", err)
		if errors.Is(err, svcLocal.ErrProtectedProcess) {
			respondError(c, http.StatusForbidden, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "终止进程失败："+err.Error())
		return
	}

	respondSuccess(c, "已发送终止信号", gin.H{"pid": pid})
}
