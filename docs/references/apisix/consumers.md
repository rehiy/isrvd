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

写操作只通过 APISIX 局部更新（`PATCH`）改动路由上的 `key-auth` 与 `consumer-restriction` 插件，不影响路由的其他字段和插件；路由不存在时返回 500。

- **认证由别处管理时**不写入也不删除路由上的 `key-auth`（`key_auth` 参数被忽略），只管理 `consumer-restriction`，清除授权时路由仍保留原有认证。判定规则：路由自身没有 `key-auth`，而路由与其引用的 PluginConfig、Service 合并后已有认证类插件（APISIX 插件元数据 `type` 为 `auth`，如 `key-auth`、`basic-auth`、`jwt-auth`、`multi-auth`，新增的认证插件自动适用）：
  - 再叠加独立 `key-auth` 会变成必须同时通过，只通过其他认证的调用方会被拦截。
  - 路由上的同名 `key-auth` 会覆盖 PluginConfig 的，改掉调用方正在使用的请求头。
- **授权由 PluginConfig 或 Service 管理时**（它们有 `consumer-restriction` 而路由自身没有）拒绝修改并返回 500，否则路由上的写入会悄悄覆盖它。
- 其他情况下 `consumer-restriction` 被写成纯白名单（原有的黑名单、自定义拒绝响应 `rejected_code`、`rejected_msg` 等配置会被覆盖）。

### 获取访问授权

```bash
isrvd_get "/apisix/whitelist"
```

返回路由自身 `consumer-restriction.whitelist` 非空的路由列表，并填充 `consumers` 字段标识授权消费者。引用的 PluginConfig 中的 `consumer-restriction` 不会列出。

### 配置/更新路由访问授权

```bash
isrvd_post "/apisix/whitelist" '{"route_id":"<ROUTE_ID>","consumers":["<USERNAME>","<NEW_USERNAME>"],"key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

用于首次配置或更新路由访问授权：写入路由侧 `key-auth` 插件，并用 `consumers` 覆盖 `consumer-restriction.whitelist`。不会创建 Consumer，`consumers` 中的用户必须已存在，且已配置 `key-auth`（认证由别处管理时所需认证方式未知，不校验）（先用 `POST /apisix/consumer` 创建）。响应 payload 为更新后的路由详情（不含 `consumers`）。

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| consumers | string[] | 授权 Consumer 用户名列表（trim 并去重）；显式传空数组 `[]` 视为删除授权，移除路由上的 consumer-restriction 和 key-auth，此时无需 `key_auth`；不传或为 `null` 返回 500（避免漏传字段时误将路由变为公开） |
| key_auth | object | 写入路由的 `key-auth` 插件配置；`consumers` 非空且认证不由别处管理时必填（先校验 `key_auth`，后校验用户名）；认证由别处管理时忽略（见上） |

`key_auth` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| header | string | 请求头认证参数名；`consumers` 非空时不能为空 |
| query | string | URL 查询认证参数名，可选 |
| hide_credentials | boolean | 是否在转发到上游前隐藏认证凭据 |

### 新建用户并加入访问授权

一次调用完成「创建 Consumer + 配置 key-auth + 加入路由访问授权」。先确认路由存在，再创建 Consumer 并更新路由，**非原子**：路由更新失败时会尽力回滚本次新建的 Consumer（已存在而被复用的不受影响）。Consumer 已存在时直接复用，不更新其 `key`，且必须已配置 `key-auth`（认证由别处管理时不校验），否则返回 500；新建的 Consumer 只配置 `key-auth`，认证由别处管理且不含 `key-auth` 时该用户无法通过认证；用户追加到路由现有白名单（已在白名单中则直接返回当前路由，不改动路由）。认证未由别处管理时，`key_auth` 会覆盖路由现有 key-auth 配置。响应 payload 为更新后的路由详情。

```bash
isrvd_post "/apisix/whitelist/user" '{"route_id":"<ROUTE_ID>","username":"<USERNAME>","key":"<AUTH_KEY>","key_auth":{"header":"token","query":"token","hide_credentials":false}}'
```

| 字段 | 类型 | 说明 |
|------|------|------|
| route_id | string | 已存在的 APISIX Route ID，不能为空 |
| username | string | 新建 Consumer 的用户名，不能为空 |
| key | string | key-auth 认证 key，不能为空 |
| key_auth | object | 写入路由的 `key-auth` 插件配置，不能为空（同上表） |
