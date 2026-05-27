package server

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	svcAgent "isrvd/internal/service/agent"

	"isrvd/config"
)

// defineAgentRoutes 定义 Agent 模块路由（LLM 代理）
func (app *App) defineAgentRoutes() []Route {
	return []Route{
		{Method: "ANY", Path: "/agent/*path", Handler: app.agentProxy, Module: "agent", Label: "代理 LLM API 请求"},
	}
}

func (app *App) agentProxy(c *gin.Context) {
	if config.Agent.BaseURL == "" {
		respondError(c, http.StatusServiceUnavailable, "Agent LLM 未配置")
		return
	}

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
