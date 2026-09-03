package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	svcAgent "isrvd/internal/service/agent"

	"isrvd/config"
)

// defineAgentRoutes 定义 Agent 模块路由（LLM 代理）
func (app *App) defineAgentRoutes() []Route {
	return []Route{
		{Method: "GET", Path: "/agent/openapi", Handler: app.agentOpenAPI, Module: "agent", Label: "查阅 OpenAPI 文档", Access: AccessAuth, IndexOnly: true},
		{Method: "ANY", Path: "/agent/*path", Handler: app.agentDispatch, Module: "agent", Label: "代理 LLM API 请求"},
		{Method: "POST", Path: "/agui", Handler: app.agentAGUI, Module: "agent", Label: "AG-UI 协议对话", QueryToken: true},
	}
}

// agentOpenAPI 按模块、路径或关键词查阅嵌入的官方 OpenAPI 文档
func (app *App) agentOpenAPI(c *gin.Context) {
	var req svcAgent.OpenAPIQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := app.agentSvc.OpenAPILookup(req)
	if err != nil {
		if errors.Is(err, svcAgent.ErrOpenAPIUnavailable) {
			respondError(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "", result)
}

// agentAGUI 以 AG-UI 协议与前端 CopilotKit 交互，响应为 SSE 事件流
func (app *App) agentAGUI(c *gin.Context) {
	var input svcAgent.AGUIInput
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

	if err := app.agentSvc.RunAGUI(c.Request.Context(), w, input); err != nil {
		logman.Error("agent agui: run failed", "error", err)
		// 响应头已提交，错误只能以 AG-UI 事件形式下发给客户端
		_ = svcAgent.WriteAGUIError(w, input, err)
	}
}

// agentDispatch 承接 /agent/*path。Gin 不能同时注册 /agent/openapi 与 /agent/*path，
// 故 GET openapi 只进权限索引，实际请求在这里分流。
func (app *App) agentDispatch(c *gin.Context) {
	if c.Request.Method == http.MethodGet && strings.TrimPrefix(c.Param("path"), "/") == "openapi" {
		app.agentOpenAPI(c)
		return
	}
	app.agentProxy(c)
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
