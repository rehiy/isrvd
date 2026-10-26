# APISIX Consumer 与访问授权 API

## Consumer 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| username | string | 消费者名（唯一标识） |
| desc | string | 描述 |
| plugins | object | 认证插件（如 key-auth, jwt-auth） |
| create_time | number | 创建时间（只读，Unix 秒；创建接口响应中为 0） |
| update_time | number | 更新时间（只读，Unix 秒；创建接口响应中为 0） |

> 列表与创建响应中，认证插件的敏感字段（`key`、`username`、`password`、`secret`、`public_key`、`key_id`、`secret_key`）会脱敏（长度 ≤ 2 整体替换为 `******`，3–10 位保留首尾各 1 位）；更新时原样回传脱敏值（含 `******`）即保留原值。

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

全量替换 `desc` 与 `plugins`（`plugins` 不传即清空）。Consumer 不存在返回 500。响应 payload 仅含 `{username, desc}`。

## 删除消费者

```bash
isrvd_delete "/apisix/consumer/<USERNAME>"
```

不会清理路由白名单中对该 Consumer 的引用。

---

## 访问授权

写操作读取路由详情时，若 APISIX 响应缺少有效 `value`，返回 500 并中止更新。

### 获取访问授权

```bash
isrvd_get "/apisix/whitelist"
```

返回 `consumer-restriction.whitelist` 非空的路由列表，并填充 `consumers` 字段标识授权消费者。

### 配置/更新路由访问授权

```bash
isrvd_post "/apisix/whitelist" '{"route_id":"<ROUTE_ID>","consumers":["<USERNAME>","<NEW_USERNAME>"],"key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

用于首次配置或更新路由访问授权：写入路由侧 `key-auth` 插件，并用 `consumers` 覆盖 `consumer-restriction.whitelist`。不会创建 Consumer，`consumers` 中的用户必须已存在（先用 `POST /apisix/consumer` 创建）。响应 payload 为更新后的路由详情（不含 `consumers`）。

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| consumers | string[] | 授权 Consumer 用户名列表（trim 并去重）；为空数组、`null` 或不传时视为删除授权，移除路由上的 consumer-restriction 和 key-auth，此时无需 `key_auth` |
| key_auth | object | 写入路由的 `key-auth` 插件配置；`consumers` 非空时必填（先校验 `key_auth`，后校验用户名） |

`key_auth` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| header | string | 请求头认证参数名；`consumers` 非空时不能为空 |
| query | string | URL 查询认证参数名，可选 |
| hide_credentials | boolean | 是否在转发到上游前隐藏认证凭据 |

### 新建用户并加入访问授权

一次调用完成「创建 Consumer + 配置 key-auth + 加入路由访问授权」。先创建 Consumer 再更新路由，**非原子**：路由更新失败时不回滚已创建的 Consumer。Consumer 已存在时直接复用，不更新其 `key`；用户追加到路由现有白名单（已在白名单中则直接返回当前路由，不改动路由）。`key_auth` 会覆盖路由现有 key-auth 配置。响应 payload 为更新后的路由详情。

```bash
isrvd_post "/apisix/whitelist/user" '{"route_id":"<ROUTE_ID>","username":"<USERNAME>","key":"<AUTH_KEY>","key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| username | string | 新建 Consumer 的用户名，不能为空 |
| key | string | key-auth 认证 key，不能为空 |
| key_auth | object | 写入路由的 `key-auth` 插件配置，不能为空（同上表） |
