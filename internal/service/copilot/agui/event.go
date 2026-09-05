// Package agui 实现 AG-UI（Agent-User Interaction）协议的事件模型。
//
// 事件结构对齐 @ag-ui/core 的类型定义；SSE 帧格式为：
//
//	data: <json>
//
// 帧之间以空行分隔，与浏览器 EventSource 及 @ag-ui/client 的解析逻辑一致。
package agui

// EventType AG-UI 事件类型
type EventType string

const (
	// RunStarted 运行开始，携带 threadId 与 runId
	RunStarted EventType = "RUN_STARTED"
	// RunFinished 运行正常结束
	RunFinished EventType = "RUN_FINISHED"
	// RunError 运行异常结束
	RunError EventType = "RUN_ERROR"
	// TextMessageStart 文本消息开始，携带 messageId 与 role
	TextMessageStart EventType = "TEXT_MESSAGE_START"
	// TextMessageContent 文本增量，delta 为追加内容
	TextMessageContent EventType = "TEXT_MESSAGE_CONTENT"
	// TextMessageEnd 文本消息结束
	TextMessageEnd EventType = "TEXT_MESSAGE_END"
	// ToolCallStart 工具调用开始，携带工具名
	ToolCallStart EventType = "TOOL_CALL_START"
	// ToolCallArgs 工具参数增量，多次到达需按序拼接
	ToolCallArgs EventType = "TOOL_CALL_ARGS"
	// ToolCallEnd 工具参数发送完毕，参数已完整
	ToolCallEnd EventType = "TOOL_CALL_END"
)

// Message AG-UI 消息，用于承载会话历史
type Message struct {
	ID         string     `json:"id"`
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string     `json:"toolCallId,omitempty"`
}

// ToolCall 助手消息中的工具调用请求
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 工具调用的函数名与参数（参数为 JSON 字符串）
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool 工具声明，parameters 为 JSON Schema
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Event AG-UI 事件。字段按事件类型选择性填充，其余字段省略。
type Event struct {
	Type EventType `json:"type"`

	// 生命周期事件：RunStarted / RunFinished / RunError
	ThreadID string `json:"threadId,omitempty"`
	RunID    string `json:"runId,omitempty"`

	// 文本消息事件：TextMessageStart / TextMessageContent / TextMessageEnd
	MessageID string `json:"messageId,omitempty"`
	Role      string `json:"role,omitempty"`
	Delta     string `json:"delta,omitempty"`

	// 工具调用事件：ToolCallStart / ToolCallArgs / ToolCallEnd
	ToolCallID   string `json:"toolCallId,omitempty"`
	ToolCallName string `json:"toolCallName,omitempty"`
	ParentMsgID  string `json:"parentMessageId,omitempty"`

	// RunError
	Error string `json:"message,omitempty"`
	Code  string `json:"code,omitempty"`
}

// RunAgentInput AG-UI 运行请求体，由 @ag-ui/client 的 HttpAgent 提交
type RunAgentInput struct {
	ThreadID string    `json:"threadId"`
	RunID    string    `json:"runId"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools"`
	// Context 前端通过 useCopilotReadable 注入的页面上下文
	Context []ContextEntry `json:"context"`
	// State 前端共享状态，透传给 LLM 的 forwardedProps
	State          map[string]any `json:"state"`
	ForwardedProps map[string]any `json:"forwardedProps"`
}

// ContextEntry 前端注入的上下文条目
type ContextEntry struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}
