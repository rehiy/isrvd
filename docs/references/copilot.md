# Copilot API

## 概述

Copilot 模块提供两类能力：

- **接口目录**（`GET /api/copilot/catalog`）：按模块、路径或关键词检索嵌入的官方 OpenAPI，供 Copilot 与脚本按需读取契约
- **AG-UI 协议对话**（`POST /api/copilot/agui`）：供前端 CopilotKit 使用，以 SSE 事件流返回 AG-UI 事件

登录即可查阅 OpenAPI，不依赖 `agent.baseUrl`，也不受 `server.openapi`（对外 Scalar 文档页）开关影响。

---

## 查阅 OpenAPI

```bash
# 模块目录
isrvd_get "/copilot/catalog"

# 按模块列出接口
isrvd_get "/copilot/catalog?tag=docker"

# 按关键词搜索
isrvd_get "/copilot/catalog?q=container"

# 查看单条接口的参数与字段（已展开 $ref）
isrvd_get "/copilot/catalog?path=/docker/containers&method=get"
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `tag` | string | 否 | 模块标签，如 `docker`、`apisix`、`caddy`、`swarm` |
| `q` | string | 否 | 关键词，匹配路径、摘要、operationId |
| `path` | string | 否 | 准确的 API 路径，如 `/docker/containers` 或 `docker/container/{id}`（也接受 `:id`）；模糊查找使用 `q` |
| `method` | string | 否 | HTTP 方法：`get` / `post` / `put` / `patch` / `delete` |

均不传时返回 `mode=catalog`（标签与接口数量）。有过滤条件且恰好命中一条时返回 `mode=detail`（参数、请求体、200 响应字段，`$ref` 已展开）；多条时返回 `mode=list`（最多 40 条，超出时 `truncated=true`）。

**响应字段：**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `mode` | string | `catalog` / `list` / `detail` |
| `hint` | string | 下一步查阅建议 |
| `tags` | array | catalog：`{name, count}` |
| `total` | number | list：匹配总数 |
| `truncated` | boolean | list：是否截断 |
| `operations` | array | list：`{method, path, summary, operationId, tag}` |
| `method` | string | detail：HTTP 方法 |
| `path` | string | detail：相对于 `/api` 的路径 |
| `summary` | string | detail：中文摘要 |
| `description` | string | detail：补充说明 |
| `operationId` | string | detail：OpenAPI operationId |
| `tag` | string | detail：模块标签 |
| `parameters` | array | detail：路径/查询参数 |
| `requestBody` | object | detail：请求体字段 |
| `requestBodyRequired` | boolean | detail：是否必须提交请求体；为 `false` 时省略 |
| `response` | object | detail：200 响应字段 |
| `toolUnsupportedReason` | string | detail 或 operations 元素：通用 Agent 工具不支持调用的原因；支持时省略 |

文档来自构建时生成的 `public/openapi/data.json`（`go run ./cmd/openapi-gen/`），与对外 `/openapi/` 页同一份规格。

SSE、WebSocket、仅支持 multipart 的上传接口，以及 `/swarm/token`、`/account/token`、`/account/2fa/totp/begin`，只提供说明，不签发 `callRef`。已有引用在刷新接口定义后也会检查限制。日志请使用同资源的非流式接口，文件上传和令牌操作请由用户在页面完成。Compose 同时支持 JSON 和 multipart，因此 JSON 部署仍可使用工具。

工具结果在返回模型和大结果暂存前统一递归脱敏：密码、secret、token、API key、JWT、authorization、私钥和 `key` 字段替换为占位符；字符串中的 PEM 私钥也会隐藏。卡片使用同一脱敏逻辑，不改变原始业务 API 的响应。该规则不识别任意自由文本中的所有凭据，仍不得用工具读取密钥文件。

---

## AG-UI 对话

```
POST /api/copilot/agui
```

**功能：** 接收 [AG-UI](https://github.com/ag-ui-protocol/ag-ui) 协议的 `RunAgentInput`，转换为 OpenAI 兼容请求发给上游 LLM，再将流式响应翻译为 AG-UI 事件以 SSE 返回。

普通成员需授予 `POST /api/copilot/agui` 权限；前端助手入口与侧栏同时检查此权限和 Agent 服务可用性。

此端点属于 CopilotKit 内部协议，不纳入通用 OpenAPI 或 `lookup_api` 目录；请求与事件格式以本节为准。

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
| `TOOL_CALL_ARGS` | 工具参数增量，按 `toolCallId` 分别顺序拼接；保留首片参数并支持并行工具分片 |
| `TOOL_CALL_END` | 工具参数发送完毕，前端据此执行工具 |
| `RUN_FINISHED` | 运行正常结束 |
| `RUN_ERROR` | 运行异常，携带 `message`，可选 `code` |

**示例：** Bash 封装的最后一个参数传 `--raw`，直接输出 SSE，不经过 JSON 解析；需要 curl 7.76.0+（`--fail-with-body`）。

```bash
isrvd_post "/copilot/agui" '{
    "threadId": "t1",
    "runId": "r1",
    "messages": [{"id": "m1", "role": "user", "content": "列出容器"}],
    "tools": [],
    "context": [{"description": "当前页面", "value": "Docker 容器列表"}]
  }' --raw
```

**上游请求说明：**

- 请求发往 `agent.baseUrl` 拼接 `/chat/completions`，并强制 `stream: true`
- 模型名使用 `agent.model`；`context` 与工具声明分别以 system 消息和 `tools` 字段注入
- 上游非 200 时返回 `RUN_ERROR` 事件，而非 HTTP 错误码（响应头已提交）

---

## 权限要求

- 查阅接口目录（`GET /api/copilot/catalog`）：登录即可
- 对话（`POST /api/copilot/agui`）：需要 `POST /api/copilot/agui` 权限

Chat iSrvd 的 API 工具保持为固定的三步流程：

1. `lookup_api` 查询上述 OpenAPI。列表和详情中的每个真实操作都会附加一个仅在当前页面内存中有效的 `callRef`，默认 30 分钟过期，刷新或离开页面后清空。
2. `isrvd_api` 或 `isrvd_mutation` 只接收 `callRef` 与业务参数，不接收 HTTP 方法和路径。业务参数使用 JSON 字符串 `{"path":{},"query":{},"body":{}}`；执行器根据引用恢复 OpenAPI 操作、校验路径参数/查询参数/请求体，再调用现有业务 REST 接口。列表查询产生的引用会在首次执行前按准确的 `path + method` 自动补全契约。
3. GET 查询由 `isrvd_api` 执行，并按返回结构展示资源列表或字段；POST、PUT、PATCH、DELETE 写操作由 `isrvd_mutation` 承接，先展示从引用解析出的真实目标和脱敏参数，只有用户在审批卡中确认后才会执行。用户取消会作为工具结果返回给 Agent。

执行失败会返回结构化 `error.kind`。引用不存在或契约变化时要求重新查询；参数错误和资源不存在时要求按契约或当前资源修正。相同 `callRef + arguments` 连续失败两次后返回 `RETRY_LIMIT`，防止模型反复重放同一错误请求。`callRef` 用于约束模型选择真实操作，登录认证与模块权限仍由后端负责。

---

## 安全说明

- `agent.apiKey` 是敏感字段，通过 `GET /api/system/config` 不会返回
- AG-UI 仅在服务端使用该密钥请求上游模型
- 建议在配置中使用环境变量或密钥管理工具存储 `apiKey`
