# Caddy 服务管理

Caddy 的 HTTP app 可以包含多个命名服务（原生配置中称为 server，默认服务为 `srv0`）。服务接口只管理监听地址、协议、超时、SNI 校验和自动 HTTPS 开关；路由等子资源由各自接口管理。

> 服务名是 Caddy `servers` map 的原始 key，创建后不可重命名。接口返回不透明 `id`，详情、更新和删除都应使用该 `id`，不要自行拼接或解析名称。

## 列表与详情

```bash
isrvd_get "/caddy/servers"
isrvd_get "/caddy/server/<ID>"
```

列表仅返回服务摘要，不返回路由内容、认证哈希、日志、TLS 策略或其他 Caddy 原生扩展配置。编辑前应通过详情接口读取最新配置。

```json
{
  "id": "~c3J2MA",
  "name": "srv0",
  "listen": [":80", ":443"],
  "protocols": ["h1", "h2", "h3"],
  "automatic_https": {
    "disable": false,
    "disable_redirects": false
  },
  "idle_timeout": "5m",
  "routeCount": 3
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 服务的不透明标识，用于详情、更新和删除 |
| `name` | string | 非空 UTF-8 字符串；作为 Caddy 原始服务名保存，可包含中文、空格、点或斜杠 |
| `listen` | string[] | 监听地址；创建和更新时至少一项 |
| `protocols` | string[] | `h1`、`h2`、`h2c`、`h3`；`h2`/`h2c` 必须与 `h1` 同时启用 |
| `automatic_https` | object | 仅包含 `disable` 与 `disable_redirects` |
| `strict_sni_host` | boolean | 严格校验 SNI Host；字段缺失表示使用 Caddy 默认规则 |
| `idle_timeout` | string | 空闲超时 |
| `read_timeout` | string | 读取超时 |
| `write_timeout` | string | 写入超时 |
| `max_header_bytes` | integer | 最大请求头字节数；`0` 表示使用默认值 |
| `routeCount` | integer | 服务下的路由数量；只读 |

## 创建

`name` 与非空 `listen` 必填。接口不会自动填入 `:80`，以免与现有服务监听地址冲突。

```bash
isrvd_post "/caddy/server" '{
  "name": "internal",
  "listen": ["127.0.0.1:8080"],
  "protocols": ["h1", "h2c"],
  "read_timeout": "10s",
  "write_timeout": "30s"
}'
```

## 更新

更新采用可编辑字段的 PUT replacement 语义：

- `listen` 必须提供且至少包含一项；
- `protocols` 缺失、`null` 或空数组时恢复 Caddy 默认协议；
- `strict_sni_host` 缺失或 `null` 时恢复 Caddy 默认规则；
- 超时缺失、`null` 或空字符串时清空；`max_header_bytes` 缺失、`null` 或 `0` 时恢复默认值；
- `automatic_https` 缺失或 `null` 时清理本接口管理的两个开关；
- 路由、日志、错误处理、TLS 策略、指标和未知扩展字段始终原样保留。

```bash
isrvd_put "/caddy/server/~aW50ZXJuYWw" '{
  "listen": ["127.0.0.1:8080"],
  "protocols": ["h1", "h2c"],
  "strict_sni_host": null
}'
```

## 删除

删除服务会同时删除其路由。结构化接口不允许删除默认服务 `srv0`，也不允许删除当前最后一个服务；需要替换默认服务结构时使用原始配置接口。

```bash
isrvd_delete "/caddy/server/<ID>"
```

## 路由归属

管理非默认服务的路由或 Basic Auth 时，将列表返回的原始 `name` 作为 `server` query 参数。名称位于 query 中，可安全包含斜杠等字符：

```bash
isrvd_get "/caddy/routes?server=internal"
isrvd_post "/caddy/route?server=internal" '{"handle":[{"handler":"static_response","body":"ok"}]}'
isrvd_get "/caddy/basic-auth?server=internal"
```

不带 `server` 时使用默认服务 `srv0`。

## HTTPS 行为

自动 HTTPS 与 HTTP→HTTPS 重定向按服务配置，不属于 Caddy 全局选项。更新服务时通过 `automatic_https` 设置：

```bash
SERVER=$(isrvd_get "/caddy/server/<ID>")
UPDATED=$(echo "$SERVER" | jq '
  del(.id, .name, .routeCount)
  | .automatic_https = {
      disable: false,
      disable_redirects: true
    }
')
isrvd_put "/caddy/server/<ID>" "$UPDATED"
```

`automatic_https` 为 `null` 或缺失时清除这两个显式开关，恢复 Caddy 对该服务的默认行为。
