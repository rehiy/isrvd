package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	svcCron "isrvd/internal/service/cron"
)

// defineCronRoutes 定义计划任务模块路由
func (app *App) defineCronRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/cron/types", Handler: app.cronTypes, Module: "cron", Label: "获取可用脚本类型", Access: AccessAuth},
		{Method: "GET", Path: "/cron/jobs", Handler: app.cronJobList, Module: "cron", Label: "列出计划任务"},
		{Method: "POST", Path: "/cron/jobs", Handler: app.cronJobCreate, Module: "cron", Label: "创建计划任务"},
		{Method: "PUT", Path: "/cron/jobs/:id", Handler: app.cronJobUpdate, Module: "cron", Label: "更新计划任务"},
		{Method: "DELETE", Path: "/cron/jobs/:id", Handler: app.cronJobDelete, Module: "cron", Label: "删除计划任务"},
		{Method: "POST", Path: "/cron/jobs/:id/run", Handler: app.cronJobRun, Module: "cron", Label: "立即执行任务"},
		{Method: "POST", Path: "/cron/jobs/:id/enable", Handler: app.cronJobEnable, Module: "cron", Label: "启用或禁用任务"},
		{Method: "GET", Path: "/cron/jobs/:id/logs", Handler: app.cronJobLogs, Module: "cron", Label: "查询任务执行历史"},
	}
}

// ─── 请求结构 ───

type cronJobEnableReq struct {
	Enabled bool `json:"enabled"`
}

type cronJobLogsReq struct {
	Limit int `form:"limit"`
}

// ─── Handler 方法 ───

func (app *App) cronTypes(c *gin.Context) {
	respondSuccess(c, "获取脚本类型成功", gin.H{"types": app.cronSvc.AvailableTypes()})
}

func (app *App) cronJobList(c *gin.Context) {
	jobs := app.cronSvc.ListJobs()
	respondSuccess(c, "获取任务列表成功", gin.H{"jobs": jobs})
}

func (app *App) cronJobCreate(c *gin.Context) {
	var req svcCron.JobUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	job, err := app.cronSvc.CreateJobFromRequest(req)
	if err != nil {
		logman.Error("Create cron job failed", "name", req.Name, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "任务创建成功", gin.H{"job": job})
}

func (app *App) cronJobUpdate(c *gin.Context) {
	id := c.Param("id")

	var req svcCron.JobUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	job, err := app.cronSvc.UpdateJobFromRequest(id, req)
	if err != nil {
		logman.Error("Update cron job failed", "id", id, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "任务更新成功", gin.H{"job": job})
}

func (app *App) cronJobDelete(c *gin.Context) {
	id := c.Param("id")
	if err := app.cronSvc.DeleteJob(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondSuccess(c, "任务删除成功", nil)
}

func (app *App) cronJobRun(c *gin.Context) {
	id := c.Param("id")
	if err := app.cronSvc.JobRun(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondSuccess(c, "任务已触发执行", nil)
}

func (app *App) cronJobEnable(c *gin.Context) {
	id := c.Param("id")

	var req cronJobEnableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := app.cronSvc.JobStatusPatch(id, req.Enabled); err != nil {
		logman.Error("Toggle cron job failed", "id", id, "enabled", req.Enabled, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "任务状态更新成功", nil)
}

func (app *App) cronJobLogs(c *gin.Context) {
	id := c.Param("id")

	var req cronJobLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	logs := app.cronSvc.JobLogs(id, req.Limit)
	respondSuccess(c, "获取执行历史成功", gin.H{"logs": logs})
}
