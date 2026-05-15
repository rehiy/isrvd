package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/strutil"

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

type cronJobUpsertReq struct {
	Name        string `json:"name" binding:"required"`
	Schedule    string `json:"schedule" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Content     string `json:"content" binding:"required"`
	WorkDir     string `json:"workDir"`
	Image       string `json:"image"`
	Container   string `json:"container"`
	Volumes     string `json:"volumes"`
	Timeout     uint   `json:"timeout"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

type cronJobEnableReq struct {
	Enabled bool `json:"enabled"`
}

type cronJobLogsReq struct {
	Limit int `form:"limit"`
}

// ─── Handler 方法 ───

func (app *App) cronTypes(c *gin.Context) {
	respondSuccess(c, "Types retrieved", gin.H{"types": app.cronSvc.AvailableTypes()})
}

func (app *App) cronJobList(c *gin.Context) {
	jobs := app.cronSvc.ListJobs()
	respondSuccess(c, "Jobs listed", gin.H{"jobs": jobs})
}

func (app *App) cronJobCreate(c *gin.Context) {
	var req cronJobUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	job := &svcCron.Job{
		ID:          strutil.NewString(),
		Name:        req.Name,
		Schedule:    req.Schedule,
		Type:        req.Type,
		Content:     req.Content,
		WorkDir:     req.WorkDir,
		Image:       req.Image,
		Container:   req.Container,
		Volumes:     req.Volumes,
		Timeout:     req.Timeout,
		Enabled:     req.Enabled,
		Description: req.Description,
	}

	if err := app.cronSvc.CreateJob(job); err != nil {
		logman.Error("Create cron job failed", "name", req.Name, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "Job created", gin.H{"job": job})
}

func (app *App) cronJobUpdate(c *gin.Context) {
	id := c.Param("id")

	var req cronJobUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	job := &svcCron.Job{
		ID:          id,
		Name:        req.Name,
		Schedule:    req.Schedule,
		Type:        req.Type,
		Content:     req.Content,
		WorkDir:     req.WorkDir,
		Image:       req.Image,
		Container:   req.Container,
		Volumes:     req.Volumes,
		Timeout:     req.Timeout,
		Enabled:     req.Enabled,
		Description: req.Description,
	}

	if err := app.cronSvc.UpdateJob(job); err != nil {
		logman.Error("Update cron job failed", "id", id, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "Job updated", gin.H{"job": job})
}

func (app *App) cronJobDelete(c *gin.Context) {
	id := c.Param("id")
	if err := app.cronSvc.DeleteJob(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondSuccess(c, "Job deleted", nil)
}

func (app *App) cronJobRun(c *gin.Context) {
	id := c.Param("id")
	if err := app.cronSvc.JobRun(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondSuccess(c, "Job triggered", nil)
}

func (app *App) cronJobEnable(c *gin.Context) {
	id := c.Param("id")

	var req cronJobEnableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := app.cronSvc.JobToggle(id, req.Enabled); err != nil {
		logman.Error("Toggle cron job failed", "id", id, "enabled", req.Enabled, "error", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "Job updated", nil)
}

func (app *App) cronJobLogs(c *gin.Context) {
	id := c.Param("id")

	var req cronJobLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	logs := app.cronSvc.JobLogs(id, limit)
	respondSuccess(c, "Logs retrieved", gin.H{"logs": logs})
}
