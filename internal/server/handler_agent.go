package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/helper"
)

func (app *App) agentProxy(c *gin.Context) {
	if config.Agent.BaseURL == "" {
		helper.RespondError(c, http.StatusServiceUnavailable, "Agent LLM 未配置")
		return
	}

	subPath := c.Param("path")
	targetURL := strings.TrimRight(config.Agent.BaseURL, "/") + subPath

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		helper.RespondError(c, http.StatusBadRequest, "读取请求体失败")
		return
	}

	rewritten := agentRewriteBody(body, config.Agent.Model)
	if !bytes.Equal(rewritten, body) {
		logman.Info("agent proxy: model rewritten", "model", config.Agent.Model)
	}
	body = rewritten

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "构造代理请求失败")
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logman.Error("agent proxy: upstream request failed", "err", err)
		helper.RespondError(c, http.StatusBadGateway, "上游 LLM 请求失败")
		return
	}
	defer resp.Body.Close()

	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
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
