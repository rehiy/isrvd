# APISIX Consumer 与访问授权 API

## Consumer 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| username | string | 消费者名（唯一标识） |
| desc | string | 描述 |
| plugins | object | 认证插件（如 key-auth, jwt-auth） |
| create_time | number | 创建时间（只读，Unix 秒；创建接口响应中为 0） |
| update_time | number | 更新时间（只读，Unix 秒；创建接口响应中为 0） |

> 列表与创建响应中，认证插件的敏感字段（`key`、`username`、`password`、`secret`、`public_key`、`key_id`、`secret_key`）会脱敏；更新时原样回传脱敏值（含 `******`）即保留原值。

## 列出消费者

```bash
isrvd_get "/apisix/consumers"
```

## 创建消费者

```bash
isrvd_post "/apisix/consumer" '{"username":"<USERNAME>","desc":"<DESC>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

`username` 必填（缺失返回 400）。`plugins` 为空时自动生成 32 位随机 key 的 `key-auth`（响应中为脱敏值）。创建调用 APISIX `PUT /consumers/<USERNAME>`，不检查重名，同名 Consumer 会被直接覆盖。

## 更新消费者

```bash
isrvd_put "/apisix/consumer/<USERNAME>" '{"desc":"<DESC>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

更新为**全量替换** `desc` 与 `plugins`，`plugins` 需完整提交（不传会被清空）。响应 payload 仅含 `{username, desc}`。

## 删除消费者

```bash
isrvd_delete "/apisix/consumer/<USERNAME>"
```

---

## 访问授权

以下写操作读取路由详情时，如果 APISIX 成功响应缺少 `value` 或其为 `null`，接口返回 HTTP 500 的错误 JSON，并停止本次路由更新。

### 获取访问授权

```bash
isrvd_get "/apisix/whitelist"
```

返回 `consumer-restriction.whitelist` 非空的路由列表，并填充 `consumers` 字段标识授权消费者。

### 配置/更新路由访问授权

该接口既用于首次配置路由访问授权，也用于更新已有授权的用户列表和路由侧 `key-auth` 参数。

如需加入不存在的 Consumer，请先创建 Consumer：

```bash
isrvd_post "/apisix/consumer" '{"username":"<NEW_USERNAME>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

再配置路由访问授权：

```bash
isrvd_post "/apisix/whitelist" '{"route_id":"<ROUTE_ID>","consumers":["<USERNAME>","<NEW_USERNAME>"],"key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

请求体字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| consumers | string[] | 已存在的授权 Consumer 用户名列表；为空数组、`null` 或不传时视为删除授权。用户名会 trim 并去重，全部为空时报"授权用户不能为空" |
| key_auth | object | 写入路由的 `key-auth` 插件配置；`consumers` 非空时必填 |

`key_auth` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| header | string | 请求头认证参数名；`consumers` 非空时不能为空 |
| query | string | URL 查询认证参数名，可选 |
| hide_credentials | boolean | 是否在转发到上游前隐藏认证凭据 |

该接口会按请求体中的 `key_auth` 写入路由侧 `key-auth` 插件，并用 `consumers` 覆盖写入 `consumer-restriction.whitelist`，用于把已有路由纳入 Consumer 访问授权管控或更新已有授权配置。接口只负责配置路由访问授权，不会创建 Consumer；`consumers` 中的用户必须已存在，如需新增用户应先调用 `/apisix/consumer` 创建并配置 `key-auth.key`。

编辑场景下 `consumers` 传空数组（或不传）则视为删除授权，将移除路由上的 consumer-restriction 和 key-auth 插件配置，此时无需 `key_auth`。

### 新建用户并加入访问授权

一次调用完成「创建 Consumer + 配置 key-auth + 加入路由访问授权」。执行顺序为先创建 Consumer 再更新路由，**非原子**：路由更新失败时已创建的 Consumer 不会回滚。Consumer 已存在时直接复用，且不会更新其 `key`。

> ⚠️ 当前实现读取已有授权名单时取的是路由详情的 `consumers` 字段（详情接口不填充该字段），因此会把路由白名单**覆盖为仅含 `<USERNAME>`**。需要保留已有授权用户时，请改用 `POST /apisix/whitelist` 显式提交完整 `consumers` 列表。

```bash
isrvd_post "/apisix/whitelist/user" '{"route_id":"<ROUTE_ID>","username":"<USERNAME>","key":"<AUTH_KEY>","key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

请求体字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| username | string | 新建 Consumer 的用户名，不能为空 |
| key | string | key-auth 认证 key，不能为空 |
| key_auth | object | 写入路由的 `key-auth` 插件配置，不能为空（同上表） |
