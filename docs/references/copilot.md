# Copilot API

## 概述

Copilot 模块提供两类能力：

- **接口目录**（`GET /api/copilot/catalog`）：按模块、路径或关键词检索嵌入的官方 OpenAPI，供 Copilot 与脚本按需读取契约
- **AG-UI 协议对话**（`POST /api/copilot/agui`）：供前端 CopilotKit 使用，以 SSE 事件流返回 AG-UI 事件

登录即可查阅 OpenAPI，不依赖 `copilot.baseUrl`，也不受 `server.openapi`（对外 Scalar 文档页）开关影响。

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
| `q` | string | 否 | 关键词，匹配路径、摘要、operationId、tag |
| `path` | string | 否 | 准确的 API 路径，如 `/docker/containers` 或 `docker/container/{id}`（也接受 `:id`）；模糊查找使用 `q` |
| `method` | string | 否 | HTTP 方法：`get` / `post` / `put` / `patch` / `delete` |

均不传时返回 `mode=catalog`（标签与接口数量）。有过滤条件且恰好命中一条时返回 `mode=detail`（参数、请求体、200 响应字段，`$ref` 已展开）；多条时返回 `mode=list`（最多 40 条，超出时 `truncated=true`）。

**响应字段：**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `mode` | string | `catalog` / `list` / `detail` |
| `hint` | string | 下一步查阅建议 |
| `tags` | array | catalog：`{name, count}` |
| `total` | number | list：匹配总数；为 0 时省略 |
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

文档来自构建时生成的 `public/openapi/data.json`（`go run ./server/cmd/openapi-gen/`），与对外 `/openapi/` 页同一份规格。

SSE、WebSocket、仅支持 multipart 的上传接口，以及 `/swarm/token`、`/account/token`、`/account/2fa/totp/begin`，只提供说明，不签发 `callRef`。已有引用在刷新接口定义后也会检查限制。日志请使用同资源的非流式接口，文件上传和令牌操作请由用户在页面完成。Compose 同时支持 JSON 和 multipart，因此 JSON 部署仍可使用工具。

工具结果在返回模型和大结果暂存前统一递归脱敏：密码、secret、token、API key、JWT、authorization、私钥和 `key` 字段替换为占位符；字符串中的 PEM 私钥也会隐藏。卡片使用同一脱敏逻辑，不改变原始业务 API 的响应。该规则不识别任意自由文本中的所有凭据，仍不得用工具读取密钥文件。

---

## AG-UI 对话

```
POST /api/copilot/agui
```

**功能：** 接收 [AG-UI](https://github.com/ag-ui-protocol/ag-ui) 协议的 `RunAgentInput`，转换为 OpenAI 兼容请求发给上游 LLM，再将流式响应翻译为 AG-UI 事件以 SSE 返回。

普通成员需授予 `POST /api/copilot/agui` 权限；前端仅在具备该权限且服务可用（`bootstrap` 的 `probe.copilot`，要求 `baseUrl` 与 `apiKey` 均已配置）时挂载 Copilot。后端对该接口仅检查 `baseUrl`。

此端点属于 CopilotKit 内部协议，不纳入通用 OpenAPI 或 `lookup_api` 目录；请求与事件格式以本节为准。

**请求头：**

| 头 | 说明 |
| --- | --- |
| `Content-Type` | `application/json` |
| `Authorization` | `Bearer <YOUR_JWT>`；SSE 无法携带头时可用 `?token=<JWT>` 代替 |

**请求体：**（读取大小受 `server.maxUploadSize` 限制，超限被截断，解析失败返回 400；未配置 `copilot.baseUrl` 时返回 503；JSON 格式错误或缺少 `runId` 返回 400）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `threadId` | string | 会话线程 ID |
| `runId` | string | 本次运行 ID，**必填** |
| `messages` | array | 会话历史，元素含 `id`、`role`、`content`；`assistant` 可带 `toolCalls`（`{id, type, function:{name, arguments}}`）；`tool` 可带 `toolCallId`。转发给上游的仅 `role`、`content`、`toolCalls`、`toolCallId`，`name` 只接收不转发 |
| `tools` | array | 前端工具声明，元素含 `name`、`description`、`parameters`（JSON Schema） |
| `context` | array | 页面上下文，元素含 `description`、`value`；`value` 为空的条目忽略，其余合并为追加在全部消息之后的一条 system 消息 |
| `state` | object | 前端共享状态；接收但不转发给上游 |
| `forwardedProps` | object | 附加属性；接收但不转发给上游 |

**响应：** `text/event-stream`，每行一个 `data:` 帧，事件类型包括：

| 事件 | 说明 |
| --- | --- |
| `RUN_STARTED` | 运行开始，携带 `threadId`、`runId` |
| `TEXT_MESSAGE_START` | 文本消息开始，携带 `messageId`（等于 `runId`）、`role` |
| `TEXT_MESSAGE_CONTENT` | 文本增量，携带 `delta` |
| `TEXT_MESSAGE_END` | 文本消息结束 |
| `TOOL_CALL_START` | 工具调用开始，携带 `toolCallId`、`toolCallName`、`parentMessageId`（等于 `runId`） |
| `TOOL_CALL_ARGS` | 工具参数增量，按 `toolCallId` 分别顺序拼接 |
| `TOOL_CALL_END` | 工具参数发送完毕，仅在上游 `finish_reason=tool_calls` 时发送 |
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

- 请求发往 `copilot.baseUrl` 拼接 `/chat/completions`，强制 `stream: true`，单次请求超时 10 分钟
- 模型名使用 `copilot.model`；`copilot.apiKey` 为空时不带 `Authorization` 头
- 上游非 200 或连接失败时不发 `RUN_STARTED`，直接发 `RUN_ERROR`（`message` 为 `上游 LLM 返回 <状态码>: <响应体前 4096 字节>`），而非 HTTP 错误码

---

## 权限要求

- 查阅接口目录（`GET /api/copilot/catalog`）：登录即可
- 对话（`POST /api/copilot/agui`）：需要 `POST /api/copilot/agui` 权限

前端 Chat iSrvd 的 API 工具为固定三步流程（前端行为）：`lookup_api` 查询上述 OpenAPI 并为操作附加仅页面内存有效、30 分钟过期的 `callRef`；`isrvd_api`（GET）或 `isrvd_mutation`（POST/PUT/PATCH/DELETE，需用户在审批卡中确认）只接收 `callRef` 与业务参数 `{"path":{},"query":{},"body":{}}`，校验后调用现有业务接口。相同 `callRef + arguments` 连续失败两次返回 `RETRY_LIMIT`；登录认证与模块权限仍由后端负责。

---

## 安全说明

- `copilot.apiKey` 是敏感字段，通过 `GET /api/system/config` 不会返回
- AG-UI 仅在服务端使用该密钥请求上游模型
- 建议在配置中使用环境变量或密钥管理工具存储 `apiKey`
