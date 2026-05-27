package server

import (
	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"
	"github.com/rehiy/libgo/websocket"
)

// defineShellRoutes 定义 Shell 模块路由（Web 终端）
func (app *App) defineShellRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/shell", Handler: app.shellWebSocket, Module: "shell", Label: "打开 Web Shell 终端"},
	}
}

func (app *App) shellWebSocket(c *gin.Context) {
	username := c.GetString("username")
	member := app.accountSvc.MemberInspect(username)
	if member == nil {
		logman.Error("用户不存在", "username", username)
		c.AbortWithStatus(403)
		return
	}

	shell := c.DefaultQuery("shell", "")

	app.wsConfig.Handler(func(conn *websocket.ServerConn) {
		app.shellSvc.RunTerminal(conn, shell, member.HomeDirectory)
	})(c)
}
