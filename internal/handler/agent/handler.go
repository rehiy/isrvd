// Package agent 提供 LLM Agent 代理转发接口
package agent

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/helper"
)

// AgentHandler 处理 Agent 相关请求
type AgentHandler struct{}

// NewAgentHandler 创建 AgentHandler 实例
func NewAgentHandler() *AgentHandler {
	return &AgentHandler{}
}

// Proxy 反向代理 LLM 请求，注入 Authorization header
func (h *AgentHandler) Proxy(c *gin.Context) {
	if config.Agent.BaseURL == "" {
		helper.RespondError(c, http.StatusServiceUnavailable, "Agent LLM 未配置")
		return
	}

	// 拼接目标 URL：baseURL + 剩余路径
	subPath := c.Param("path")
	targetURL := strings.TrimRight(config.Agent.BaseURL, "/") + subPath

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	// 构造上游请求
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, strings.NewReader(string(body)))
	if err != nil {
		logman.Error("agent proxy: build request failed", "err", err)
		helper.RespondError(c, http.StatusInternalServerError, "构造代理请求失败")
		return
	}

	// 透传请求头（排除 Host 和 Authorization）
	for key, vals := range c.Request.Header {
		k := strings.ToLower(key)
		if k == "host" || k == "authorization" {
			continue
		}
		for _, v := range vals {
			req.Header.Add(key, v)
		}
	}

	// 注入 Authorization
	if config.Agent.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.Agent.APIKey)
	}

	// 透传查询参数
	req.URL.RawQuery = c.Request.URL.RawQuery

	// 发起请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logman.Error("agent proxy: upstream request failed", "err", err)
		helper.RespondError(c, http.StatusBadGateway, "上游 LLM 请求失败")
		return
	}
	defer resp.Body.Close()

	// 透传响应头
	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}

	// 流式透传响应体
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}
