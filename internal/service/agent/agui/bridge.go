// Package agui 将 OpenAI 兼容的流式响应转换为 AG-UI 事件流。
package agui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamDone 是 OpenAI 流式响应的结束标记
const StreamDone = "[DONE]"

// ChatRequest OpenAI 聊天补全请求体
type ChatRequest struct {
	Model    string        `json:"model,omitempty"`
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
	Tools    []ChatTool    `json:"tools,omitempty"`
}

// ChatMessage OpenAI 消息
type ChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	Name       string         `json:"name,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
}

// ChatTool OpenAI 工具声明
type ChatTool struct {
	Type     string       `json:"type"`
	Function ChatToolFunc `json:"function"`
}

// ChatToolFunc OpenAI 工具函数声明
type ChatToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ChatToolCall OpenAI 工具调用
type ChatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ChatToolCallFunc `json:"function"`
}

// ChatToolCallFunc OpenAI 工具调用的函数名与参数
type ChatToolCallFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ChatChunk OpenAI 流式响应的单个 data 载荷
type ChatChunk struct {
	ID      string        `json:"id"`
	Choices []ChunkChoice `json:"choices"`
	Error   *ChunkError   `json:"error,omitempty"`
}

// ChunkChoice 流式响应的 choice 项
type ChunkChoice struct {
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

// ChunkDelta 流式增量，可能同时携带内容与工具调用
type ChunkDelta struct {
	Role      string         `json:"role,omitempty"`
	Content   string         `json:"content,omitempty"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"`
}

// ChunkError 上游返回的错误载荷
type ChunkError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// RunOptions AG-UI 运行参数
type RunOptions struct {
	// Endpoint 上游 chat/completions 地址
	Endpoint string
	// APIKey 上游鉴权密钥
	APIKey string
	// Model 模型名，覆盖请求体中的 model
	Model string
	// Timeout 单次请求超时
	Timeout time.Duration
}

// Run 执行一次 AG-UI 运行：将输入转换为 OpenAI 请求，并把流式响应翻译为 AG-UI 事件。
//
// 事件写入 enc；heartbeat 非零时按该间隔写出注释行以保持连接活跃。
func Run(ctx context.Context, enc *Encoder, input RunAgentInput, opt RunOptions) error {
	body, err := json.Marshal(buildChatRequest(input, opt.Model))
	if err != nil {
		return fmt.Errorf("构造上游请求失败: %w", err)
	}

	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, opt.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造上游请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if opt.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+opt.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求上游 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("上游 LLM 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	return translate(ctx, enc, resp.Body, input)
}

// buildChatRequest 将 AG-UI 输入映射为 OpenAI 请求。
// Context 条目与工具结果以 system 消息注入，使页面上下文与工具返回对模型可见。
func buildChatRequest(input RunAgentInput, model string) ChatRequest {
	req := ChatRequest{Stream: true}
	if model != "" {
		req.Model = model
	}

	for _, m := range input.Messages {
		switch m.Role {
		case "tool":
			req.Messages = append(req.Messages, ChatMessage{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		case "assistant":
			msg := ChatMessage{Role: "assistant", Content: m.Content}
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, ChatToolCall{
					ID:       tc.ID,
					Type:     "function",
					Function: ChatToolCallFunc{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
				})
			}
			req.Messages = append(req.Messages, msg)
		default:
			req.Messages = append(req.Messages, ChatMessage{Role: m.Role, Content: m.Content})
		}
	}

	if ctxText := renderContext(input.Context); ctxText != "" {
		req.Messages = append(req.Messages, ChatMessage{Role: "system", Content: ctxText})
	}

	for _, t := range input.Tools {
		req.Tools = append(req.Tools, ChatTool{
			Type: "function",
			Function: ChatToolFunc{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	return req
}

// renderContext 将前端注入的上下文条目渲染为单条 system 消息
func renderContext(entries []ContextEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("以下是当前应用提供的上下文信息：\n")
	for _, e := range entries {
		if e.Value == "" {
			continue
		}
		sb.WriteString("\n- ")
		sb.WriteString(e.Description)
		sb.WriteString("：")
		sb.WriteString(e.Value)
	}
	return sb.String()
}

// translate 读取 OpenAI SSE 流并转换为 AG-UI 事件
func translate(ctx context.Context, enc *Encoder, body io.Reader, input RunAgentInput) error {
	if err := enc.Write(Event{Type: RunStarted, ThreadID: input.ThreadID, RunID: input.RunID}); err != nil {
		return err
	}

	// toolIDs 记录已发送 TOOL_CALL_START 的工具，用于识别参数分片所属的工具
	toolIDs := map[string]bool{}
	messageStarted := false
	messageID := input.RunID
	lastToolID := ""

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == StreamDone {
			continue
		}

		var chunk ChatChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// 单个分片解析失败不应中断整条流
			continue
		}
		if chunk.Error != nil {
			return enc.Write(Event{
				Type:     RunError,
				ThreadID: input.ThreadID,
				RunID:    input.RunID,
				Error:    chunk.Error.Message,
				Code:     chunk.Error.Code,
			})
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			if !messageStarted {
				if err := enc.Write(Event{
					Type:      TextMessageStart,
					MessageID: messageID,
					Role:      "assistant",
				}); err != nil {
					return err
				}
				messageStarted = true
			}
			if err := enc.Write(Event{
				Type:      TextMessageContent,
				MessageID: messageID,
				Delta:     delta.Content,
			}); err != nil {
				return err
			}
		}

		for _, tc := range delta.ToolCalls {
			// 流式首片带 ID 与函数名，后续分片仅携带参数增量且 ID 为空，
			// 故以最近一次出现的工具 ID 归属参数分片。
			if tc.ID != "" {
				toolIDs[tc.ID] = true
				lastToolID = tc.ID
				if messageStarted {
					if err := enc.Write(Event{Type: TextMessageEnd, MessageID: messageID}); err != nil {
						return err
					}
					messageStarted = false
				}
				if err := enc.Write(Event{
					Type:         ToolCallStart,
					ToolCallID:   tc.ID,
					ToolCallName: tc.Function.Name,
					ParentMsgID:  messageID,
				}); err != nil {
					return err
				}
				continue
			}
			if tc.Function.Arguments != "" && lastToolID != "" {
				if err := enc.Write(Event{
					Type:       ToolCallArgs,
					ToolCallID: lastToolID,
					Delta:      tc.Function.Arguments,
				}); err != nil {
					return err
				}
			}
		}

		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "tool_calls" {
			// 工具参数已发送完毕，逐个补发 TOOL_CALL_END，前端据此执行工具
			for id := range toolIDs {
				if err := enc.Write(Event{Type: ToolCallEnd, ToolCallID: id}); err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return fmt.Errorf("读取上游流式响应失败: %w", err)
	}

	if messageStarted {
		if err := enc.Write(Event{Type: TextMessageEnd, MessageID: messageID}); err != nil {
			return err
		}
	}

	return enc.Write(Event{Type: RunFinished, ThreadID: input.ThreadID, RunID: input.RunID})
}
