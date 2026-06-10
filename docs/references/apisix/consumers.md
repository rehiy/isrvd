# APISIX Consumer 与白名单 API

## Consumer 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| username | string | 消费者名（唯一标识） |
| desc | string | 描述 |
| plugins | object | 认证插件（如 key-auth, jwt-auth） |

## 列出消费者

```bash
isrvd_get "/apisix/consumers"
```

## 创建消费者

```bash
isrvd_post "/apisix/consumer" '{"username":"<USERNAME>","desc":"<DESC>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

## 更新消费者

```bash
isrvd_put "/apisix/consumer/<USERNAME>" '{"desc":"<DESC>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

## 删除消费者

```bash
isrvd_delete "/apisix/consumer/<USERNAME>"
```

---

## 白名单

### 获取白名单

```bash
isrvd_get "/apisix/whitelist"
```

返回含 key-auth + consumer-restriction 的路由列表，`consumers` 字段标识白名单消费者。

### 配置路由白名单

如需加入不存在的 Consumer，请先创建 Consumer：

```bash
isrvd_post "/apisix/consumer" '{"username":"<NEW_USERNAME>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
```

再配置路由白名单：

```bash
isrvd_post "/apisix/whitelist" '{"route_id":"<ROUTE_ID>","consumers":["<USERNAME>","<NEW_USERNAME>"],"key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

请求体字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| consumers | string[] | 已存在的白名单 Consumer 用户名列表，不能为空 |
| key_auth | object | 写入路由的 `key-auth` 插件配置，不能为空 |

`key_auth` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| header | string | 请求头认证参数名，不能为空 |
| query | string | URL 查询认证参数名，可选 |
| hide_credentials | boolean | 是否在转发到上游前隐藏认证凭据 |

该接口会按请求体中的 `key_auth` 写入路由侧 `key-auth` 插件，并写入 `consumer-restriction.whitelist`，用于把已有路由纳入 Consumer 白名单管控。接口只负责配置路由白名单，不会创建 Consumer；`consumers` 中的用户必须已存在，如需新增用户应先调用 `/apisix/consumer` 创建并配置 `key-auth.key`。

### 新建用户并加入白名单（原子接口）

一次性完成「创建 Consumer + 配置 key-auth + 加入路由白名单」三步操作，若 Consumer 已存在则直接复用（幂等）：

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

### 撤销白名单

```bash
isrvd_delete "/apisix/whitelist/user/<ROUTE_ID>/<USERNAME>"
```

该接口会从指定路由的 `consumer-restriction.whitelist` 中移除指定 Consumer，路径参数分别为路由 ID 和 Consumer 用户名。
