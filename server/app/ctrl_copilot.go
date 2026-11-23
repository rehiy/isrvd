package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	"isrvd/server/config"
	"isrvd/server/service/copilot"
)

// defineCopilotRoutes 定义 Copilot 聊天与接口目录路由。
func (app *App) defineCopilotRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/copilot/catalog", Handler: app.copilotCatalog, Module: "copilot", Label: "查阅接口目录", Access: AccessAuth},
		{Method: "POST", Path: "/copilot/agui", Handler: app.copilotAGUI, Module: "copilot", Label: "Copilot 对话"},
	}
}

// copilotCatalog 按模块、路径或关键词查阅嵌入的官方 OpenAPI 文档。
func (app *App) copilotCatalog(c *gin.Context) {
	var req copilot.OpenAPIQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.copilotSvc.OpenAPILookup(req)
	if err != nil {
		if errors.Is(err, copilot.ErrOpenAPIUnavailable) {
			respondError(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

// decodeAGUIInput 严格读取单个 JSON 请求，尺寸上限也覆盖 JSON 后的内容。
func decodeAGUIInput(w http.ResponseWriter, r *http.Request, maxBytes int64, input *copilot.AGUIInput) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	if err := decoder.Decode(input); err != nil {
		return err
	}
	var extra json.RawMessage
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("请求体包含多个 JSON 值")
	}
	return err
}

// copilotAGUI 以 AG-UI 协议与前端 CopilotKit 交互，响应为 SSE 事件流。
func (app *App) copilotAGUI(c *gin.Context) {
	var input copilot.AGUIInput
	if err := decodeAGUIInput(c.Writer, c.Request, config.Current().Server.MaxUploadSize, &input); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			respondError(c, http.StatusRequestEntityTooLarge, "请求体过大")
		} else {
			respondError(c, http.StatusBadRequest, "请求体格式错误")
		}
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

	if err := app.copilotSvc.RunAGUI(c.Request.Context(), w, input); err != nil && !errors.Is(err, context.Canceled) {
		logman.Error("copilot agui: run failed", "error", err)
		_ = copilot.WriteAGUIError(w, input, err)
	}
}
