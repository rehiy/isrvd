# Agent 代理 API

## 概述

Agent 模块提供两类能力：

- **LLM API 代理**（`ANY /api/agent/*path`）：转发到配置的 OpenAI 兼容 API，自动注入 `agent.apiKey` 并可重写 `agent.model`
- **AG-UI 协议对话**（`POST /api/agui`）：供前端 CopilotKit 使用，以 SSE 事件流返回 AG-UI 事件

---

## AG-UI 对话

```
POST /api/agui
```

**功能：** 接收 [AG-UI](https://github.com/ag-ui-protocol/ag-ui) 协议的 `RunAgentInput`，转换为 OpenAI 兼容请求发给上游 LLM，再将流式响应翻译为 AG-UI 事件以 SSE 返回。

**请求头：**

| 头 | 说明 |
| --- | --- |
| `Content-Type` | `application/json` |
| `Authorization` | `Bearer <YOUR_JWT>`；SSE 无法携带头时可用 `?token=<JWT>` 代替 |

**请求体：**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `threadId` | string | 会话线程 ID |
| `runId` | string | 本次运行 ID，**必填** |
| `messages` | array | 会话历史，元素含 `id`、`role`、`content`；`tool` 角色可带 `toolCallId` |
| `tools` | array | 前端工具声明，元素含 `name`、`description`、`parameters`（JSON Schema） |
| `context` | array | 前端注入的页面上下文，元素含 `description`、`value`，会被合并为 system 消息 |
| `forwardedProps` | object | 透传给上游的附加属性 |

**响应：** `text/event-stream`，每行一个 `data:` 帧，事件类型包括：

| 事件 | 说明 |
| --- | --- |
| `RUN_STARTED` | 运行开始，携带 `threadId`、`runId` |
| `TEXT_MESSAGE_START` | 文本消息开始，携带 `messageId`、`role` |
| `TEXT_MESSAGE_CONTENT` | 文本增量，携带 `delta` |
| `TEXT_MESSAGE_END` | 文本消息结束 |
| `TOOL_CALL_START` | 工具调用开始，携带 `toolCallId`、`toolCallName` |
| `TOOL_CALL_ARGS` | 工具参数增量，需按序拼接 |
| `TOOL_CALL_END` | 工具参数发送完毕，前端据此执行工具 |
| `RUN_FINISHED` | 运行正常结束 |
| `RUN_ERROR` | 运行异常，携带 `error` |

**示例：**

```bash
curl -X POST "http://<HOST>/api/agui" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_JWT>" \
  -d '{
    "threadId": "t1",
    "runId": "r1",
    "messages": [{"id": "m1", "role": "user", "content": "列出容器"}],
    "tools": [],
    "context": [{"description": "当前页面", "value": "Docker 容器列表"}]
  }'
```

**上游请求说明：**

- 请求发往 `agent.baseUrl` 拼接 `/chat/completions`，并强制 `stream: true`
- 模型名使用 `agent.model`；`context` 与工具声明分别以 system 消息和 `tools` 字段注入
- 上游非 200 时返回 `RUN_ERROR` 事件，而非 HTTP 错误码（响应头已提交）

---

## 代理请求

```
ANY /api/agent/*path  (代理到配置的 OpenAI 兼容 API)
```

**配置要求：**

在 `config.yml` 中配置 `agent` 段：

```yaml
agent:
  model: "gpt-3.5-turbo"          # 默认模型
  baseUrl: "https://api.openai.com/v1"  # API 基础 URL
  apiKey: "sk-..."                  # API 密钥（敏感，GET 不返回）
```

**行为说明：**

1. 所有 `/api/agent/*` 的请求都会被代理到 `agent.baseUrl` 对应的地址
2. 自动在请求头中添加 `Authorization: Bearer <agent.apiKey>`
3. 如果请求体中指定了 `model` 字段，使用请求中的值；否则使用 `agent.model`
4. 支持所有 HTTP 方法（GET/POST/PUT/DELETE 等）
5. 请求体大小受 `server.maxUploadSize` 限制

**示例：**

```bash
# 聊天补全
curl -X POST "http://<HOST>/api/agent/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_JWT>" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# 列出模型
curl -X GET "http://<HOST>/api/agent/models" \
  -H "Authorization: Bearer <YOUR_JWT>"
```

---

## 权限要求

- 需要登录（任意认证方式）
- 需要 `agent` 模块权限

---

## 前端集成

前端可以通过 `/api/agent/` 路径直接调用 LLM API，无需在客户端暴露 API Key：

```javascript
const response = await fetch('/api/agent/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${jwtToken}`
  },
  body: JSON.stringify({
    model: 'gpt-3.5-turbo',
    messages: [{ role: 'user', content: 'Hello!' }]
  })
});

const data = await response.json();
console.log(data.choices[0].message.content);
```

---

## 安全说明

- `agent.apiKey` 是敏感字段，通过 `GET /api/system/config` 不会返回
- 只有具有 `agent` 模块权限的用户才能使用代理功能
- 建议在配置中使用环境变量或密钥管理工具存储 `apiKey`
