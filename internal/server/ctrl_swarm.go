package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	svcSwarm "isrvd/internal/service/swarm"
)

// defineSwarmRoutes 定义 Swarm 模块路由
func (app *App) defineSwarmRoutes() []Route {
	return []Route{
		// Swarm 信息
		{Method: "GET", Path: "/swarm/info", Handler: app.swarmInfo, Module: "swarm", Label: "获取 Swarm 集群信息"},
		// 节点管理
		{Method: "GET", Path: "/swarm/nodes", Handler: app.swarmNodeList, Module: "swarm", Label: "查询 Swarm 节点列表"},
		{Method: "GET", Path: "/swarm/node/:id", Handler: app.swarmNodeInspect, Module: "swarm", Label: "获取 Swarm 节点详情"},
		{Method: "POST", Path: "/swarm/node/:id/action", Handler: app.swarmNodeAction, Module: "swarm", Label: "执行 Swarm 节点操作"},
		{Method: "GET", Path: "/swarm/token", Handler: app.swarmJoinToken, Module: "swarm", Label: "获取 Swarm 加入令牌"},
		// 服务管理
		{Method: "GET", Path: "/swarm/services", Handler: app.swarmServiceList, Module: "swarm", Label: "查询 Swarm 服务列表"},
		{Method: "GET", Path: "/swarm/service/:id", Handler: app.swarmServiceInspect, Module: "swarm", Label: "获取 Swarm 服务详情"},
		{Method: "POST", Path: "/swarm/service", Handler: app.swarmServiceCreate, Module: "swarm", Label: "创建 Swarm 服务"},
		{Method: "POST", Path: "/swarm/service/:id/action", Handler: app.swarmServiceAction, Module: "swarm", Label: "执行 Swarm 服务操作"},
		{Method: "POST", Path: "/swarm/service/:id/force-update", Handler: app.swarmServiceForceUpdate, Module: "swarm", Label: "强制更新 Swarm 服务"},
		{Method: "GET", Path: "/swarm/service/:id/logs", Handler: app.swarmServiceLogs, Module: "swarm", Label: "获取 Swarm 服务日志"},
		// 任务
		{Method: "GET", Path: "/swarm/tasks", Handler: app.swarmTaskList, Module: "swarm", Label: "查询 Swarm 任务列表"},
	}
}

func (app *App) swarmInfo(c *gin.Context) {
	result, err := app.swarmSvc.Info(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取 Swarm 信息成功", result)
}

func (app *App) swarmNodeList(c *gin.Context) {
	result, err := app.swarmSvc.NodeList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取节点列表成功", result)
}

func (app *App) swarmNodeInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.swarmSvc.NodeInspect(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取节点详情成功", result)
}

func (app *App) swarmNodeAction(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.swarmSvc.NodeAction(c.Request.Context(), c.Param("id"), req.Action); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "节点操作成功", nil)
}

func (app *App) swarmServiceList(c *gin.Context) {
	result, err := app.swarmSvc.ServiceList(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取服务列表成功", result)
}

func (app *App) swarmServiceInspect(c *gin.Context) {
	id := c.Param("id")
	result, err := app.swarmSvc.ServiceInspect(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取服务详情成功", result)
}

func (app *App) swarmServiceCreate(c *gin.Context) {
	var req svcSwarm.ServiceSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := app.swarmSvc.ServiceCreate(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "服务创建成功", gin.H{"id": id})
}

func (app *App) swarmServiceAction(c *gin.Context) {
	var req struct {
		Action   string  `json:"action"`
		Replicas *uint64 `json:"replicas,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.swarmSvc.ServiceAction(c.Request.Context(), c.Param("id"), req.Action, req.Replicas); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "服务操作成功", nil)
}

func (app *App) swarmServiceForceUpdate(c *gin.Context) {
	if err := app.swarmSvc.ServiceForceUpdate(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "服务强制更新成功", nil)
}

func (app *App) swarmServiceLogs(c *gin.Context) {
	serviceID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")
	logs, err := app.swarmSvc.ServiceLogs(c.Request.Context(), serviceID, tail)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取日志成功", gin.H{"logs": logs})
}

func (app *App) swarmTaskList(c *gin.Context) {
	serviceID := c.Query("serviceID")
	result, err := app.swarmSvc.TaskList(c.Request.Context(), serviceID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取任务列表成功", result)
}

func (app *App) swarmJoinToken(c *gin.Context) {
	result, err := app.swarmSvc.JoinToken(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "获取加入令牌成功", result)
}
