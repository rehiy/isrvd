package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	svcCopilot "isrvd/internal/service/copilot"
)

// defineCopilotRoutes 定义 Copilot 聊天与接口目录路由。
func (app *App) defineCopilotRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/copilot/catalog", Handler: app.copilotCatalog, Module: "copilot", Label: "查阅接口目录", Access: AccessAuth},
		{Method: "POST", Path: "/copilot/agui", Handler: app.copilotAGUI, Module: "copilot", Label: "Copilot 对话", QueryToken: true},
	}
}

// copilotCatalog 按模块、路径或关键词查阅嵌入的官方 OpenAPI 文档。
func (app *App) copilotCatalog(c *gin.Context) {
	var req svcCopilot.OpenAPIQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.copilotSvc.OpenAPILookup(req)
	if err != nil {
		if errors.Is(err, svcCopilot.ErrOpenAPIUnavailable) {
			respondError(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

// copilotAGUI 以 AG-UI 协议与前端 CopilotKit 交互，响应为 SSE 事件流。
func (app *App) copilotAGUI(c *gin.Context) {
	var input svcCopilot.AGUIInput
	if err := json.NewDecoder(io.LimitReader(c.Request.Body, config.Server.MaxUploadSize)).Decode(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if input.RunID == "" {
		respondError(c, http.StatusBadRequest, "缺少 runId")
		return
	}

	w, err := httpd.NewEventWriter(c.Writer)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := app.copilotSvc.RunAGUI(c.Request.Context(), w, input); err != nil {
		logman.Error("copilot agui: run failed", "error", err)
		_ = svcCopilot.WriteAGUIError(w, input, err)
	}
}
