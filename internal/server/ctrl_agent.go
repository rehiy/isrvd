package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

// defineAgentRoutes 定义 Agent 模块路由（LLM 代理）
func (app *App) defineAgentRoutes() []Route {
	return []Route{
		{Method: "ANY", Path: "/agent/*path", Handler: app.agentProxy, Module: "agent", Label: "代理 LLM API 请求"},
	}
}

var agentHTTPClient = &http.Client{Timeout: 10 * time.Minute}

func (app *App) agentProxy(c *gin.Context) {
	if config.Agent.BaseURL == "" {
		respondError(c, http.StatusServiceUnavailable, "Agent LLM 未配置")
		return
	}

	subPath := c.Param("path")
	targetURL := strings.TrimRight(config.Agent.BaseURL, "/") + subPath

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	rewritten := agentRewriteBody(body, config.Agent.Model)
	if !bytes.Equal(rewritten, body) {
		logman.Info("agent proxy: model rewritten", "model", config.Agent.Model)
	}
	body = rewritten

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "构造代理请求失败")
		return
	}

	for key, vals := range c.Request.Header {
		k := strings.ToLower(key)
		if k == "host" || k == "authorization" || k == "content-length" {
			continue
		}
		for _, v := range vals {
			req.Header.Add(key, v)
		}
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	if config.Agent.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.Agent.APIKey)
	}
	req.URL.RawQuery = c.Request.URL.RawQuery

	resp, err := agentHTTPClient.Do(req)
	if err != nil {
		logman.Error("agent proxy: upstream request failed", "error", err)
		respondError(c, http.StatusBadGateway, "上游 LLM 请求失败")
		return
	}
	defer resp.Body.Close()

	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logman.Error("agent proxy: stream copy failed", "error", err)
	}
}

func agentRewriteBody(body []byte, model string) []byte {
	if model == "" || len(body) == 0 {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	if _, ok := payload["model"]; !ok {
		return body
	}
	payload["model"] = model
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return rewritten
}
