// Package agent 提供 LLM Agent 代理业务服务
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/logman"

	"isrvd/config"
	"isrvd/internal/service/agent/agui"
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
	Method   string      // HTTP 方法
	SubPath  string      // 转发到上游的子路径
	RawQuery string      // 原始查询字符串
	Headers  http.Header // 透传的请求头
	Body     []byte      // 请求体原始字节
}

// ProxyResponse 代理响应
type ProxyResponse struct {
	StatusCode int           // 上游响应状态码
	Headers    http.Header   // 上游响应头
	Body       io.ReadCloser // 上游响应体（流式，调用方负责关闭）
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
		if k == "host" || k == "authorization" || k == "content-length" || k == "cookie" {
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

// AGUIInput 是 /api/agui 接受的请求体，等价于 agui.RunAgentInput
type AGUIInput = agui.RunAgentInput

// RunAGUI 以 AG-UI 协议执行一轮对话：将输入转换为 OpenAI 兼容请求，
// 并把上游流式响应翻译为 AG-UI 事件写入 w。
func (s *Service) RunAGUI(ctx context.Context, w io.Writer, input AGUIInput) error {
	return agui.Run(ctx, agui.NewEncoder(w), input, agui.RunOptions{
		Endpoint: strings.TrimRight(config.Agent.BaseURL, "/") + "/chat/completions",
		APIKey:   config.Agent.APIKey,
		Model:    config.Agent.Model,
		Timeout:  httpClient.Timeout,
	})
}

// WriteAGUIError 向已提交响应头的 SSE 流补发 RUN_ERROR 事件。
// 用于运行开始后失败的场景，此时无法再改用 HTTP 错误码。
func WriteAGUIError(w io.Writer, input AGUIInput, err error) error {
	return agui.NewEncoder(w).Write(agui.Event{
		Type:     agui.RunError,
		ThreadID: input.ThreadID,
		RunID:    input.RunID,
		Error:    err.Error(),
	})
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
