# Caddy Basic Auth API

> 管理 Caddy 路由的 HTTP Basic 认证账号。
>
> Basic Auth 通过在路由 handle 链首部插入 `authentication` handler（`http_basic` provider）实现。
> 密码以 bcrypt hash 存储；账号随路由配置保存，不跨路由共享。
>
> **只读接口**：该接口只列出已含 `authentication` handler 的路由，
> 若要为新路由启用 Basic Auth，需先通过路由编辑接口手动添加 handler，或直接调用添加账号接口（路由尚无 handler 时自动插入）。

## 列出 Basic Auth 路由

```bash
isrvd_get "/caddy/basic-auth"
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
| `realm` | — | 留空时沿用已有 realm；首次添加时设置 |
| `forwardHeader` | — | 非空时注入 `headers` handler，将 `{http.auth.user.id}` 写入该请求头转发给上游；空表示移除 |

- 若路由尚无 `authentication` handler，自动在 handle 链首部插入。

## 删除账号

```bash
isrvd_delete "/caddy/basic-auth/0/users/alice"
```

- 若删除后账号列表为空，自动移除 `authentication` handler 及透传 `headers` handler。

## 更新配置

> 仅更新已认证路由的 `realm` 与 `forwardHeader`，不修改账号列表。

```bash
isrvd_put "/caddy/basic-auth/0/config" '{
  "realm": "New Realm",
  "forwardHeader": "X-Remote-User"
}'
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `realm` | — | HTTP Basic realm |
| `forwardHeader` | — | 透传用户名的请求头名；空表示关闭透传 |
