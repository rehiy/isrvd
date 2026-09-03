package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	svcAgent "isrvd/internal/service/agent"

	"isrvd/config"
)

// defineAgentRoutes 定义 Agent 模块路由（LLM 代理）
func (app *App) defineAgentRoutes() []Route {
	return []Route{
		{Method: "ANY", Path: "/agent/*path", Handler: app.agentProxy, Module: "agent", Label: "代理 LLM API 请求"},
		{Method: "POST", Path: "/agui", Handler: app.agentAGUI, Module: "agent", Label: "AG-UI 协议对话", QueryToken: true},
	}
}

// agentAGUI 以 AG-UI 协议与前端 CopilotKit 交互，响应为 SSE 事件流
func (app *App) agentAGUI(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Server.MaxUploadSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	var input svcAgent.AGUIInput
	if err := json.Unmarshal(body, &input); err != nil {
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

	if err := app.agentSvc.RunAGUI(c.Request.Context(), w, input); err != nil {
		logman.Error("agent agui: run failed", "error", err)
		// 响应头已提交，错误只能以 AG-UI 事件形式下发给客户端
		_ = svcAgent.WriteAGUIError(w, input, err)
	}
}

func (app *App) agentProxy(c *gin.Context) {
	if config.Agent.BaseURL == "" {
		respondError(c, http.StatusServiceUnavailable, "Agent LLM 未配置")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Server.MaxUploadSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	resp, err := app.agentSvc.Proxy(svcAgent.ProxyRequest{
		Method:   c.Request.Method,
		SubPath:  c.Param("path"),
		RawQuery: c.Request.URL.RawQuery,
		Headers:  c.Request.Header,
		Body:     body,
	})
	if err != nil {
		logman.Error("agent proxy: upstream request failed", "error", err)
		respondError(c, http.StatusBadGateway, "上游 LLM 请求失败")
		return
	}
	defer resp.Body.Close()

	for key, vals := range resp.Headers {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logman.Error("agent proxy: stream copy failed", "error", err)
	}
}
