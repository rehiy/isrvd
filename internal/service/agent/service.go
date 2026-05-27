// Package agent 提供 LLM Agent 代理业务服务
package agent

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
)

var httpClient = &http.Client{Timeout: 10 * time.Minute}

// Service Agent 代理业务服务
type Service struct {
	client *http.Client
}

// NewService 创建 Agent 代理业务服务
func NewService() *Service {
	return &Service{
		client: httpClient,
	}
}

// ProxyRequest 代理请求参数
type ProxyRequest struct {
	Method   string
	SubPath  string
	RawQuery string
	Headers  http.Header
	Body     []byte
}

// ProxyResponse 代理响应
type ProxyResponse struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
}

// Proxy 转发请求到上游 LLM，并自动注入 APIKey 和 model 覆盖。
// 返回上游原始响应，由调用方负责流式转发和关闭 Body。
func (s *Service) Proxy(req ProxyRequest) (*ProxyResponse, error) {
	targetURL := strings.TrimRight(config.Agent.BaseURL, "/") + req.SubPath

	body := rewriteBody(req.Body, config.Agent.Model)
	if !bytes.Equal(body, req.Body) {
		logman.Info("agent proxy: model rewritten", "model", config.Agent.Model)
	}

	httpReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, vals := range req.Headers {
		k := strings.ToLower(key)
		if k == "host" || k == "authorization" || k == "content-length" {
			continue
		}
		for _, v := range vals {
			httpReq.Header.Add(key, v)
		}
	}
	httpReq.ContentLength = int64(len(body))
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(body)))
	if config.Agent.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+config.Agent.APIKey)
	}
	httpReq.URL.RawQuery = req.RawQuery

	resp, err := s.client.Do(httpReq)
	if err != nil {
		logman.Error("agent proxy: upstream request failed", "error", err)
		return nil, err
	}

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
	}, nil
}

// rewriteBody 将请求体中的 model 字段替换为配置的 model（若配置非空）
func rewriteBody(body []byte, model string) []byte {
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
