# Caddy 概览、全局选项与原始配置 API

## 概览信息

```bash
isrvd_get "/caddy/info"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| servers | number | server 总数 |
| routes | number | 所有 server 路由总数 |
| certs | number | SSL 证书总数，包含配置证书与 Caddy 运行时已签发缓存证书 |
| hasTls | boolean | 是否配置了 TLS app |
| available | boolean | Admin API 是否可达 |

## 全局选项

### 查询全局选项

```bash
isrvd_get "/caddy/global"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| logLevel | string | 全局日志级别：`DEBUG` / `INFO` / `WARN` / `ERROR` |
| logFormat | string | 日志格式：`json` / `console`，留空使用默认 |
| email | string | ACME 注册邮箱 |
| acmeCA | string | 自定义 ACME 目录 URL，留空使用 Let's Encrypt |
| localCerts | boolean | 使用本地自签证书（`internal` issuer），不走 ACME |
| onDemandTLS | boolean | 启用 on_demand TLS，连接时动态申请证书（同时设置全局 `permission` 和默认策略的 `on_demand: true`） |
| onDemandAsk | string | ask 鉴权端点 URL，Caddy 申请证书前向此发 GET 请求，返回 2xx 才允许；留空时仍写入 `permission.module=http` 但不含 `endpoint`，Caddy v2.8+ 须配置，否则可能被拒 |
| gracePeriod | string | HTTP app 优雅关闭等待时间，例如 `10s` |

> 所有字段均为 `omitempty`，GET 响应中空值字段会被省略。isrvd 不校验 `logLevel` / `logFormat` 取值，非法值由 Caddy 拒绝。`localCerts=true` 时忽略 `email` / `acmeCA`；仅 `onDemandTLS=true` 时写入 `onDemandAsk`，关闭 on_demand 时保留原有 permission 配置。

> 自动 HTTPS 与 HTTP→HTTPS 重定向属于 HTTP server 配置，不属于全局选项。请通过服务接口的 `automatic_https` 字段按服务设置，参见 [服务管理](servers.md)。

### 更新全局选项

```bash
# 先 GET 当前值，再修改后整体提交
GLOBAL=$(isrvd_get "/caddy/global")
isrvd_put "/caddy/global" "$(echo "$GLOBAL" | jq '.email = "you@example.com" | .logLevel = "WARN"')"
```

> ⚠️ `PUT /caddy/global` 为**全量替换**：未提交的字段会被清空（如日志级别、日志格式、ACME issuer、on_demand 开关、`gracePeriod`）。只传 `{"logLevel":"WARN"}` 会清掉 `email` / `acmeCA` / `gracePeriod` 等。

## 获取完整配置

```bash
isrvd_get "/caddy/config"
```

返回值是 Caddy 完整 JSON 配置（结构与 `GET /config/` 返回一致），可作为整体替换的输入。

> **`caddy` 镜像说明**：默认配置模板位于 `build/docker-caddy/caddy.json`，首次启动会复制并持久化到 `/data/conf/caddy.json`；Caddy 运行数据（自动签发证书、autosave 等）持久化在 `/data/caddy/`。
> 默认模板监听 `:80` 与 `:443`，并设置 `automatic_https.disable=true`、`disable_redirects=true`，避免首次启动时自动签发证书或跳转 HTTPS。
> 结构化接口（全局选项、服务、路由、证书）只改动自己管理的字段，其余配置原样保留。

## 整体替换配置

```bash
isrvd_post "/caddy/config" "$(jq -n '{
  config: {
    admin: {listen: "127.0.0.1:2019"},
    apps: {
      http: {
        servers: {
          srv0: {
            listen: [":80"],
            automatic_https: {disable_redirects: true},
            routes: []
          }
        }
      }
    }
  }
}')"
```

请求体 `{ config: <caddy-json> }`，等价于 `POST /load`，**整体原子替换**当前运行配置。`config` 必填且必须是 JSON 对象，否则 400；Caddy 拒绝时透传其 4xx，5xx 或传输失败返回 502。

> ⚠️ 这是危险操作，会影响所有正在运行的路由；推荐先用 `isrvd_get "/caddy/config"` 备份再修改。

## 状态码

| 状态码 | 场景 |
|---|---|
| 400 | 服务名或配置非法；Basic Auth、证书、config 加载接口的未归类错误 |
| 404 | 服务或路由不存在，`index` 越界 |
| 409 | 服务重名；删除 `srv0` 或最后一个服务 |
| 500 | 服务、路由接口的未归类错误 |
| 502 / 504 | Caddy 返回 5xx 或传输失败 / 超时；Caddy 返回 4xx 时透传 |

## 典型工作流

### 启用 HTTPS 自动签发

```bash
# 设置全局 ACME 邮箱（全量替换，需基于当前值修改）
isrvd_put "/caddy/global" "$(isrvd_get "/caddy/global" | jq '.email = "you@example.com"')"
```

按服务启用自动 HTTPS 与重定向见 [服务管理](servers.md)。

### 备份并修改部分字段

```bash
# 1. 备份当前配置
CFG=$(isrvd_get "/caddy/config")

# 2. 修改后提交（启用自动 HTTPS 与重定向）
isrvd_post "/caddy/config" "$(echo "$CFG" | jq '{config: (.apps.http.servers.srv0.automatic_https |= (.disable = false | .disable_redirects = false))}')"
```

### 健康检查

```bash
isrvd_get "/caddy/info" '.available'   # 应返回 true
isrvd_get "/caddy/routes"              # 路由数大于 0 表示正常工作
```
