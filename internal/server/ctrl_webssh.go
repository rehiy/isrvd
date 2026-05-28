package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/websocket"

	svcWebSSH "isrvd/internal/service/webssh"
)

// defineWebSSHRoutes 定义 WebSSH 模块路由
func (app *App) defineWebSSHRoutes() []Route {
	return []Route{
		// 主机管理
		{Method: "GET", Path: "/ssh/hosts", Handler: app.websshHostList, Module: "ssh", Label: "查询 SSH 主机列表"},
		{Method: "GET", Path: "/ssh/host/:id", Handler: app.websshHostInspect, Module: "ssh", Label: "获取 SSH 主机详情"},
		{Method: "POST", Path: "/ssh/host", Handler: app.websshHostCreate, Module: "ssh", Label: "添加 SSH 主机"},
		{Method: "PUT", Path: "/ssh/host/:id", Handler: app.websshHostUpdate, Module: "ssh", Label: "更新 SSH 主机"},
		{Method: "DELETE", Path: "/ssh/host/:id", Handler: app.websshHostDelete, Module: "ssh", Label: "删除 SSH 主机"},
		// SSH 终端
		{Method: "GET", Path: "/ssh/to/:id", Handler: app.websshTerminal, Module: "ssh", Label: "打开 SSH 终端", QueryToken: true},
	}
}

// svcWebSSHHostUpsertRequest 是 service/webssh 中 HostUpsertRequest 的本地别名
type svcWebSSHHostUpsertRequest = svcWebSSH.HostUpsertRequest

func (app *App) websshHostList(c *gin.Context) {
	respondSuccess(c, "", app.websshSvc.HostList())
}

func (app *App) websshHostInspect(c *gin.Context) {
	id := c.Param("id")
	host := app.websshSvc.HostInspect(id)
	if host == nil {
		respondError(c, http.StatusNotFound, "主机不存在")
		return
	}
	respondSuccess(c, "", host)
}

func (app *App) websshHostCreate(c *gin.Context) {
	var req svcWebSSHHostUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	host, err := app.websshSvc.HostCreate(&req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机添加成功", host)
}

func (app *App) websshHostUpdate(c *gin.Context) {
	id := c.Param("id")
	var req svcWebSSHHostUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	host, err := app.websshSvc.HostUpdate(id, &req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机更新成功", host)
}

func (app *App) websshHostDelete(c *gin.Context) {
	id := c.Param("id")
	if err := app.websshSvc.HostDelete(id); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "SSH 主机删除成功", nil)
}

func (app *App) websshTerminal(c *gin.Context) {
	id := c.Param("id")
	app.wsConfig.Handler(func(conn *websocket.ServerConn) {
		app.websshSvc.RunTerminal(conn, id)
	})(c)
}
