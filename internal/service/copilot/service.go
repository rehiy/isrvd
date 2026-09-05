package copilot

import (
	"context"
	"io"
	"strings"
	"time"

	"isrvd/config"
	"isrvd/internal/service/copilot/agui"
)

// Service Copilot 聊天与接口目录服务。
type Service struct {
	spec *openAPISpec // 仅 initServices 赋值一次，之后只读
}

// NewService 创建 Copilot 服务。
func NewService() *Service {
	return &Service{}
}

// AGUIInput 是 /api/copilot/agui 接受的请求体，等价于 agui.RunAgentInput。
type AGUIInput = agui.RunAgentInput

// RunAGUI 以 AG-UI 协议执行一轮对话：将输入转换为 OpenAI 兼容请求，
// 并把上游流式响应翻译为 AG-UI 事件写入 w。
func (s *Service) RunAGUI(ctx context.Context, w io.Writer, input AGUIInput) error {
	return agui.Run(ctx, agui.NewEncoder(w), input, agui.RunOptions{
		Endpoint: strings.TrimRight(config.Agent.BaseURL, "/") + "/chat/completions",
		APIKey:   config.Agent.APIKey,
		Model:    config.Agent.Model,
		Timeout:  10 * time.Minute,
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
