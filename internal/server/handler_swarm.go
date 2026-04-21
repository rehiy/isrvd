package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/internal/helper"
	pkgswarm "isrvd/pkgs/swarm"
)

func (app *App) swarmInfo(c *gin.Context) {
	result, err := app.swarmSvc.SwarmInfo(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Swarm info retrieved", result)
}

func (app *App) swarmListNodes(c *gin.Context) {
	result, err := app.swarmSvc.ListNodes(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Nodes listed", result)
}

func (app *App) swarmInspectNode(c *gin.Context) {
	id := c.Param("id")
	result, err := app.swarmSvc.InspectNode(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Node inspected", result)
}

func (app *App) swarmNodeAction(c *gin.Context) {
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.swarmSvc.NodeAction(c.Request.Context(), req.ID, req.Action); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Node updated", nil)
}

func (app *App) swarmListServices(c *gin.Context) {
	result, err := app.swarmSvc.ListServices(c.Request.Context())
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Services listed", result)
}

func (app *App) swarmInspectService(c *gin.Context) {
	id := c.Param("id")
	result, err := app.swarmSvc.InspectService(c.Request.Context(), id)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Service inspected", result)
}

func (app *App) swarmCreateService(c *gin.Context) {
	var req pkgswarm.SwarmCreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := app.swarmSvc.CreateService(c.Request.Context(), req)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Service created", gin.H{"id": id})
}

func (app *App) swarmServiceAction(c *gin.Context) {
	var req struct {
		ID       string  `json:"id"`
		Action   string  `json:"action"`
		Replicas *uint64 `json:"replicas,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.swarmSvc.ServiceAction(c.Request.Context(), req.ID, req.Action, req.Replicas); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Service "+req.Action+" successfully", nil)
}

func (app *App) swarmForceUpdateService(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.swarmSvc.ForceUpdateService(c.Request.Context(), req.ID); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Service force updated", nil)
}

func (app *App) swarmServiceLogs(c *gin.Context) {
	serviceID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")
	logs, err := app.swarmSvc.GetServiceLogs(c.Request.Context(), serviceID, tail)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Logs retrieved", gin.H{"logs": logs})
}

func (app *App) swarmListTasks(c *gin.Context) {
	serviceID := c.Query("serviceID")
	result, err := app.swarmSvc.ListTasks(c.Request.Context(), serviceID)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.RespondSuccess(c, "Tasks listed", result)
}
