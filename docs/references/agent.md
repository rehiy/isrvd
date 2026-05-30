# Agent 代理 API

## 概述

Agent 模块提供 OpenAI 兼容的 LLM API 代理功能，自动注入 `agent.apiKey` 并可重写 `agent.model`。

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
