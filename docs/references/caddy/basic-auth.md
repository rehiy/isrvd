# Caddy Basic Auth API

> 管理 Caddy 路由的 HTTP Basic 认证账号。
>
> Basic Auth 通过在路由 handle 链首部插入 `authentication` handler（`http_basic` provider）实现。
> 密码以 bcrypt hash 存储；账号随路由配置保存，不跨路由共享。
>
> 所有接口的 `index` 为路由下标；非默认服务在路径后追加 `?server=<服务 name>`（取服务列表中的 `name`，不是 `id`）。
>
> `GET /caddy/basic-auth` 只列出已含 `authentication` handler 的路由；为新路由启用 Basic Auth，直接调用「添加账号」即可（自动在 handle 链首插入）。
>
> 错误：账号已存在/不存在、路由未配置 Basic Auth 返回 400；下标越界或服务不存在返回 404。

## 列出 Basic Auth 路由

```bash
isrvd_get "/caddy/basic-auth"
isrvd_get "/caddy/basic-auth?server=internal"
```

响应示例：

```json
[
  {
    "index": 0,
    "name": "my-route",
    "realm": "Protected Area",
    "forwardHeader": "X-Remote-User",
    "users": [
      { "username": "alice" },
      { "username": "bob" }
    ],
    "handlers": [
      { "handler": "reverse_proxy", "upstreams": [{ "dial": "localhost:8080" }] }
    ]
  }
]
```

| 字段 | 说明 |
|---|---|
| `index` | 路由下标（主键，用于后续操作） |
| `name` | 路由 `@id`，仅展示用 |
| `realm` | HTTP Basic 认证 realm（弹窗提示文字） |
| `forwardHeader` | 认证用户名透传的请求头名；空表示未开启 |
| `users` | 账号列表（不含密码 hash） |
| `handlers` | 其余 handler 链（排除 authentication，只读展示） |

> 没有匹配路由时 payload 为 `null`；路由无其他 handler 时 `handlers` 也为 `null`。
>
> 添加账号、删除账号、更新配置都会重建整个 `authentication` handler，原 handler 中的其他字段（如 `hash` 设置、其他 provider）不会保留。

## 添加账号

```bash
isrvd_post "/caddy/basic-auth/0/users" '{
  "username": "alice",
  "password": "secret123",
  "realm": "Protected Area",
  "forwardHeader": "X-Remote-User"
}'
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `username` | ✅ | 用户名，同一路由内唯一 |
| `password` | ✅ | 明文密码，后端自动 bcrypt |
| `realm` | — | 留空时已有 handler 沿用原 realm，新建则使用 Caddy 默认值 |
| `forwardHeader` | — | 非空时注入 `headers` handler，将 `{http.auth.user.id}` 写入该请求头转发给上游；空表示移除 |

## 删除账号

```bash
isrvd_delete "/caddy/basic-auth/0/users/alice"
```

删除后账号列表为空时，自动移除 `authentication` handler 及透传 `headers` handler。

## 更新配置

仅更新已认证路由的 `realm` 与 `forwardHeader`，不修改账号列表。

```bash
isrvd_put "/caddy/basic-auth/0/config" '{
  "realm": "New Realm",
  "forwardHeader": "X-Remote-User"
}'
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `realm` | — | HTTP Basic realm；留空表示清除（使用 Caddy 默认值），与添加账号时「留空沿用」不同 |
| `forwardHeader` | — | 透传用户名的请求头名；空表示关闭透传 |
