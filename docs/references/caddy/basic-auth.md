# Caddy Basic Auth API

> 管理 Caddy 路由的 HTTP Basic 认证账号。
>
> Basic Auth 通过在路由 handle 链首部插入 `authentication` handler（`http_basic` provider）实现。
> 密码以 bcrypt hash 存储；账号随路由配置保存，不跨路由共享。
>
> 所有接口的 `index` 为快照中的路由下标，并非稳定 ID。非默认服务在路径后追加 `?server=<服务 name>`（取服务列表中的 `name`，不是 `id`）。
>
> **全部写接口必须携带非空 `revision`**：添加账号、更新配置在 JSON 请求体中传入，删除账号通过查询参数传入。缺失返回 428，版本过期返回 409，均不加载配置。旧的无版本写请求必须先读取快照后升级；默认 GET 数组响应保持兼容，仅新增只读 `revision` 字段。
>
> 错误：账号已存在/不存在、路由未配置 Basic Auth、编辑清空账号返回 400；下标越界或服务不存在返回 404；批量移除缺少 DELETE 权限返回 403。

## 列出 Basic Auth 路由

```bash
isrvd_get "/caddy/basic-auth"
isrvd_get "/caddy/basic-auth?server=internal"
isrvd_get "/caddy/basic-auth?server=internal&snapshot=true"
```

默认响应示例：

```json
[
  {
    "index": 0,
    "revision": "<opaque-revision>",
    "name": "my-route",
    "realm": "Protected Area",
    "forwardHeader": "X-Remote-User",
    "users": [{ "username": "alice" }, { "username": "bob" }],
    "handlers": [
      { "handler": "reverse_proxy", "upstreams": [{ "dial": "localhost:8080" }] }
    ]
  }
]
```

| 字段 | 说明 |
|---|---|
| `index` | 路由在 `revision` 对应快照中的下标 |
| `revision` | 只读，所属服务全部路由内容和顺序的版本；每个列表项来自同一次配置读取 |
| `name` | 路由 `@id`，仅展示用 |
| `realm` | HTTP Basic 认证 realm（弹窗提示文字） |
| `forwardHeader` | 认证用户名透传的请求头名；空表示未开启 |
| `users` | 账号列表（不含密码或 hash） |
| `handlers` | 其余 handler 链（排除 authentication，只读展示） |

`snapshot=true` 返回 `{ "routes": [...], "revision": "<opaque-revision>" }`。没有认证路由时 `routes` 为 `null`，仍返回有效版本；默认列表此时为 `null`。路由无其他 handler 时 `handlers` 也为 `null`。

快照版本与 `GET /caddy/routes?snapshot=true` 使用相同算法，覆盖该服务的全部路由，而非仅认证路由。为尚未认证的路由添加首个账号，应从路由快照中选择下标并回传该快照版本。需要同时使用两份列表时，必须验证其 `revision` 相等后才能关联下标，不能通过再次读取后直接沿用旧下标来消除竞态。

任一写操作成功后，版本都会变化。继续修改必须重新读取快照并重新选择目标；失败后应刷新列表并清理旧草稿，不可直接重试旧版本。版本在 iSrvd 的 `ConfigMutate` 修改临界区内原子校验，校验、修改、一次 `POST /load` 串行执行；不代替其他 Caddy Admin API 客户端的并发写入协调。

> 添加账号、删除账号、更新配置都会重建整个 `authentication` handler，原 handler 中的其他字段（如 `hash` 设置、其他 provider）不会保留。

## 添加账号

已认证路由使用认证快照，尚未认证的路由使用路由快照；以下示例为新路由启用认证：

```bash
SNAPSHOT=$(isrvd_get "/caddy/routes?server=srv0&snapshot=true")
REVISION=$(printf '%s' "$SNAPSHOT" | jq -r '.revision')
# 检查 SNAPSHOT.routes 后，选择当前下标 0；明文密码由调用方安全提供
isrvd_post "/caddy/basic-auth/0/users?server=srv0" \
  "$(jq -n --arg revision "$REVISION" --arg password "$BASIC_AUTH_PASSWORD" \
    '{username: "alice", password: $password, realm: "Protected Area", forwardHeader: "X-Remote-User", revision: $revision}')"
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `username` | 是 | 用户名，同一路由内唯一 |
| `password` | 是 | 明文密码，后端自动 bcrypt |
| `revision` | 是 | 所属服务路由快照版本，不能使用其他服务的列表下标 |
| `realm` | 否 | 留空时已有 handler 沿用原 realm，新建则使用 Caddy 默认值 |
| `forwardHeader` | 否 | 非空时注入 `headers` handler，将 `{http.auth.user.id}` 写入该请求头转发给上游；空表示移除 |

## 删除账号

```bash
SNAPSHOT=$(isrvd_get "/caddy/basic-auth?server=srv0&snapshot=true")
REVISION=$(printf '%s' "$SNAPSHOT" | jq -r '.revision')
# 核对快照中的下标及账号；用户名作为路径段时必须 URL 编码
isrvd_delete "/caddy/basic-auth/0/users/alice?server=srv0&revision=$REVISION"
```

删除需要 `DELETE /api/caddy/basic-auth/:index/users/:username` 权限。删除最后一个账号将同时移除 `authentication` handler 和用户名透传 handler，**解除此路由的 HTTP Basic 认证**；任何人都可能访问原受保护的资源。API 自动化调用方必须自行确认这一危险行为。

Web 界面的桌面和移动账号删除入口均先确认删除；删除最后账号还必须进入单独的危险确认步骤，并勾选知情确认后才能提交。编辑配置不允许清空最后账号，应使用此独立删除入口解除认证。

## 原子更新配置及移除账号

省略 `removeUsers` 时仅更新已认证路由的 `realm` 与 `forwardHeader`。指定 `removeUsers` 时，配置更新与所有账号移除通过一次配置加载原子提交，任何账号不存在、版本过期、加载失败都不会先提交部分配置。

```bash
SNAPSHOT=$(isrvd_get "/caddy/basic-auth?server=srv0&snapshot=true")
REVISION=$(printf '%s' "$SNAPSHOT" | jq -r '.revision')
isrvd_put "/caddy/basic-auth/0/config?server=srv0" \
  "$(jq -n --arg revision "$REVISION" \
    '{realm: "New Realm", forwardHeader: "X-Remote-User", revision: $revision, removeUsers: ["bob"]}')"
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `realm` | 否 | HTTP Basic realm；留空表示清除（使用 Caddy 默认值），与添加账号时「留空沿用」不同 |
| `forwardHeader` | 否 | 透传用户名的请求头名；空表示关闭透传 |
| `revision` | 是 | 本次编辑所使用的同服务路由快照版本 |
| `removeUsers` | 否 | 要移除的用户名数组；所有用户名必须存在，移除后至少保留一个账号 |

此操作始终需要 `PUT /api/caddy/basic-auth/:index/config` 权限。`removeUsers` 非空时，服务端在任何配置加载前额外验证 `DELETE /api/caddy/basic-auth/:index/users/:username` 权限，不能仅凭 PUT 权限移除账号。没有删除权限时，Web 编辑器的账号标签不可移除，仍可编辑 realm 与透传头。
