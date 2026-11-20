# Caddy 服务管理

Caddy 的 HTTP app 可以包含多个命名服务（原生配置中称为 server，默认服务为 `srv0`）。服务接口只管理监听地址、协议、超时、SNI 校验、可信代理和自动 HTTPS 开关；路由等子资源由各自接口管理。

> 服务名是 Caddy `servers` map 的原始 key，创建后不可重命名。接口返回不透明 `id`，详情、更新和删除都应使用该 `id`，不要自行拼接或解析名称。

## 列表与详情

```bash
isrvd_get "/caddy/servers"
isrvd_get "/caddy/server/<ID>"
```

列表（按 `name` 排序）与详情返回相同的服务视图，均不含路由、日志、TLS 策略等 Caddy 原生扩展配置。编辑前建议重新 GET 详情获取最新值。

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
| `listen` | string[] | 监听地址；创建和更新时至少一项，每项 trim 后不能为空且不能重复 |
| `protocols` | string[] | `h1`、`h2`、`h2c`、`h3`，不能重复；`h2`/`h2c` 必须与 `h1` 同时启用 |
| `automatic_https` | object | 仅包含 `disable` 与 `disable_redirects`；未设置时不返回该字段 |
| `strict_sni_host` | boolean | 严格校验 SNI Host；字段缺失表示使用 Caddy 默认规则 |
| `idle_timeout` | string | 空闲超时 |
| `read_timeout` | string | 读取超时 |
| `write_timeout` | string | 写入超时 |
| `max_header_bytes` | integer | 最大请求头字节数；`0` 表示使用默认值，负数返回 400 |
| `trusted_proxies` | object | 支持 `{"source":"static","ranges":["192.0.2.10/32"]}`；IP 或 CIDR 数组，最多 256 个，不允许重复及全网段 `/0` |
| `trusted_proxies_strict` | integer | `0` 关闭；正整数开启从右向左解析 `X-Forwarded-For`，已有大于 `1` 的解析层数设置在编辑服务时保留（具体支持取决于 Caddy 版本）；仅在配置可信代理时有效 |
| `trustedProxiesCustom` | boolean | 只读；true 表示当前使用非 static 或带扩展字段的可信代理模块，需通过原始配置维护 |
| `routeCount` | integer | 服务下的路由数量；只读 |

三个超时均为 duration 字符串（如 `5m`）；Caddy 中数字形式（纳秒）在响应里会转换为字符串。

## 创建

`name` 与非空 `listen` 必填。接口不会自动填入 `:80`，以免与现有服务监听地址冲突。重名返回 409，非法配置返回 400。

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
- `trusted_proxies` 缺失、`null` 或 `ranges` 为空时清空内置 static 可信代理列表及严格解析开关；提交时只允许 `source=static`，单个 IP 自动转为 /32 或 /128 网段；
- 若服务已有非 static 模块或带扩展字段的可信代理配置，接口不会覆盖它；要编辑该模块请使用原始配置接口；
- 路由、日志、错误处理、TLS 策略、指标和未知扩展字段始终原样保留。

```bash
isrvd_put "/caddy/server/~aW50ZXJuYWw" '{
  "listen": ["127.0.0.1:8080"],
  "protocols": ["h1", "h2c"],
  "strict_sni_host": null
}'
```

## 可信代理 IP

在「Caddy → 服务管理 → 编辑服务」设置可信代理地址，适用于 Caddy 前面有 CDN、负载均衡器或其他反向代理的场景。**仅将实际受信任的直连代理 IP 加入列表**，不要添加任意客户端地址。默认情况下 Caddy 不信任代理头，且仅从 `X-Forwarded-For` 提取客户端 IP；代理逐级追加该请求头时建议开启严格解析。界面以开关控制常规严格模式，已有的高级层数设置不会被无关编辑重置。

```bash
SERVER=$(isrvd_get "/caddy/server/<ID>")
UPDATED=$(echo "$SERVER" | jq '
  .trusted_proxies = {source: "static", ranges: ["192.0.2.10", "10.1.0.0/16"]} |
  .trusted_proxies_strict = 1
')
isrvd_put "/caddy/server/<ID>" "$UPDATED"
```

上例中的 `192.0.2.10` 将规范化为 `192.0.2.10/32`。同时支持 IPv6 单地址和 CIDR；不接受 `0.0.0.0/0` 或 `::/0`。清空名单会关闭对代理请求头的信任。该配置写入 Caddy 原生 `apps.http.servers.<name>.trusted_proxies`，与系统配置里的登录代理 `tha.trustedCIDRs` 无关。

已有 Caddy 配置使用自定义代理来源（例如插件模块）时，页面只提示，不会覆盖原模块及其严格解析设置。API 尝试通过此服务接口直接替换自定义来源会返回 400，须使用 Caddy 原始配置接口。其他未暴露的 `client_ip_headers` 等字段保持原样。

参考：[Caddy 官方 trusted_proxies 文档](https://caddyserver.com/docs/caddyfile/options#trusted_proxies)。

## 删除

删除服务会同时删除其路由。不允许删除默认服务 `srv0` 或当前最后一个服务（409），服务不存在返回 404；需要替换默认服务结构时使用原始配置接口。

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

自动 HTTPS 与 HTTP→HTTPS 重定向按服务配置，不属于 Caddy 全局选项。更新服务时通过 `automatic_https` 设置（`id`、`name`、`routeCount` 在更新时被忽略）：

```bash
SERVER=$(isrvd_get "/caddy/server/<ID>")
UPDATED=$(echo "$SERVER" | jq '
  .automatic_https = {
      disable: false,
      disable_redirects: true
    }
')
isrvd_put "/caddy/server/<ID>" "$UPDATED"
```

`automatic_https` 为 `null` 或缺失时清除这两个显式开关，恢复 Caddy 对该服务的默认行为。
