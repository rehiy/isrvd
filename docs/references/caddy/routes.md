# Caddy 路由 API

> Caddy 路由模型与 APISIX 不同：路由是 `apps.http.servers.<name>.routes` 数组中的元素，
> 用数组下标（index）作为主键。默认服务为 `srv0`，可通过 query 参数 `?server=<服务 name>` 切换（取服务列表中的 `name`）。
>
> **接口直接使用 Caddy 原生 JSON 结构**，后端不校验 `match` / `handle` 内容，透传给 Caddy，非法配置由 Caddy 在 `/load` 时拒绝并透传错误。`index` 非整数返回 400；下标越界或服务不存在返回 404。

> **执行顺序**：同一服务的路由按数组顺序执行，精确域名不会自动优先于通配符。前面的 `reverse_proxy`、`static_response` 等处理器响应后，后续路由不再执行，即使没有设置 `terminal: true`。例如 `app.example.com` 的业务路由应放在覆盖它的 `*.example.com` 兜底路由之前，并保留鉴权、重写等中间件的必要顺序。

## 列出路由

```bash
isrvd_get "/caddy/routes"                      # 默认 server=srv0
isrvd_get "/caddy/routes?server=srv0"          # 显式指定服务名称
isrvd_get "/caddy/routes?server=srv0&snapshot=true"  # 列表与版本快照
```

默认返回路由数组。`snapshot=true` 时返回以下对象，空列表同样返回有效版本：

| 字段 | 类型 | 说明 |
|------|------|------|
| routes | Route[] | 按实际执行顺序排列的路由数组，字段见下表 |
| revision | string | 只读的列表版本，覆盖当前服务路由的内容和顺序；插入、移动时必须回传，不写入 Caddy 配置 |

| 字段 | 类型 | 说明 |
|------|------|------|
| index | number | 数组下标（路由主键，只读，由后端附加）；更新时需去掉该字段，否则会作为扩展字段写入 Caddy 配置 |
| group | string | 路由分组（可选） |
| match | MatchSet[] | 匹配条件数组，元素间为 OR 关系，同一元素内字段为 AND |
| handle | Handler[] | 处理器数组，按顺序执行 |
| terminal | boolean | 终止后续路由匹配 |
| @id | string \| number | Caddy `@id` 字段（可选，用于 admin api 寻址） |

### MatchSet 字段（Caddy 原生）

| 字段 | 类型 | 说明 |
|------|------|------|
| host | string[] | 匹配 Host，留空表示不限制 |
| path | string[] | 匹配 Path，支持 `*` 通配符 |
| method | string[] | 匹配 HTTP 方法 |
| header | object | 按请求头匹配，key 为头字段名，value 为匹配值列表；value 为空数组表示只要该头存在即匹配 |
| protocol | string | 匹配协议：`http` / `https`，留空不限制 |

### Handler 字段（Caddy 原生）

每个 handler 必须包含 `handler` 字段（模块名），其余字段视模块而定。

| handler 值 | 说明 |
|---|---|
| `reverse_proxy` | 反向代理 |
| `file_server` | 静态文件服务 |
| `static_response` | 静态响应 |
| `rewrite` | URI 重写 |
| `headers` | 请求头/响应头操作 |

## 查看路由详情

```bash
isrvd_get "/caddy/route/<INDEX>"
isrvd_get "/caddy/route/0?server=srv0"
```

## 创建路由

```bash
# 反向代理
isrvd_post "/caddy/route" '{
  "match": [{"host": ["api.example.com"], "path": ["/v1/*"]}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}]
}'

# 反向代理（含超时）
isrvd_post "/caddy/route" '{
  "match": [{"host": ["api.example.com"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "backend:8080"}],
    "transport": {"protocol": "http", "dial_timeout": "10s", "response_header_timeout": "30s"}
  }]
}'

# 按协议匹配（HTTP 跳转）
isrvd_post "/caddy/route" '{
  "match": [{"protocol": "http"}],
  "handle": [{"handler": "static_response", "status_code": 301, "headers": {"Location": ["https://{http.request.host}{http.request.uri}"]}}],
  "terminal": true
}'

# 按请求头匹配
isrvd_post "/caddy/route" '{
  "match": [{"header": {"X-Internal": []}}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "internal-svc:8080"}]}]
}'

# 反向代理转发前重写 URI
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "backend:8080"}],
    "rewrite": {"strip_path_prefix": "/api"}
  }]
}'

# 设置响应头（CORS / 安全头）+ 反向代理
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [
    {
      "handler": "headers",
      "response": {"set": {"Access-Control-Allow-Origin": ["*"], "X-Frame-Options": ["SAMEORIGIN"]}}
    },
    {"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}
  ]
}'

# 透传真实 IP（请求头）
isrvd_post "/caddy/route" '{
  "handle": [{
    "handler": "headers",
    "request": {
      "set": {
        "X-Real-IP": ["{http.request.remote.host}"],
        "X-Forwarded-For": ["{http.request.remote.host}"]
      }
    }
  }]
}'

# 静态文件
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/static/*"]}],
  "handle": [{"handler": "file_server", "root": "/var/www"}]
}'

# 静态响应
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/health"]}],
  "handle": [{"handler": "static_response", "status_code": 200, "body": "OK"}]
}'

# URI 重写
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/api/*"]}],
  "handle": [{"handler": "rewrite", "strip_path_prefix": "/api"}]
}'

# 完整 URI 替换（支持占位符）
isrvd_post "/caddy/route" '{
  "match": [{"path": ["/old/*"]}],
  "handle": [{"handler": "rewrite", "uri": "/new/{http.request.uri.path.1}"}]
}'

# FastCGI（PHP-FPM）
isrvd_post "/caddy/route" '{
  "match": [{"host": ["php.example.com"]}],
  "handle": [{
    "handler": "reverse_proxy",
    "upstreams": [{"dial": "php-fpm:9000"}],
    "transport": {"protocol": "fastcgi", "root": "/var/www/html"}
  }]
}'
```

返回值 `payload.index`：新路由在数组中的下标。

### 指定插入位置

未传 `index` 时保持兼容，追加到末尾。传入 `index` 时，必须同时传入快照中的 `revision`；`index=0` 表示最前，`index=当前路由数量` 表示末尾，其余位置插入到原有该下标路由之前。

```bash
# 一次读取同时获取路由与版本，检查 routes 后选择插入位置
SNAPSHOT=$(isrvd_get "/caddy/routes?server=srv0&snapshot=true")
REVISION=$(printf '%s' "$SNAPSHOT" | jq -r '.revision')
isrvd_post "/caddy/route?server=srv0&index=0&revision=$REVISION" '{
  "match": [{"host": ["app.example.com"]}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "app:8080"}]}]
}'
```

位置超出 `0..路由数量` 返回 400；缺少版本返回 428，列表已变化返回 409，此时重新读取并确认位置后再提交。失败不会加载配置。插入成功后，原位置及后续路由的下标都会变化，应重新查询列表。

Web 界面的新建弹窗默认推荐将精确域名放在覆盖它的通配符或无 Host 限制的兜底之前。推荐仅针对单组纯 Host 匹配、单一反向代理/文件服务/静态响应处理器，且前移不会越过复杂路由的场景；不处理多组匹配、路径/方法等附加条件、鉴权、重写、路由分组或扩展处理器，不自动重排已有配置。复杂场景可手动指定插入位置。

## 调整执行顺序

```bash
SNAPSHOT=$(isrvd_get "/caddy/routes?server=srv0&snapshot=true")
REVISION=$(printf '%s' "$SNAPSHOT" | jq -r '.revision')
# 将当前下标 2 的路由移到下标 0
isrvd_put "/caddy/route/2/position?server=srv0" \
  "$(jq -n --arg revision "$REVISION" '{index: 0, revision: $revision}')"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| index | number | 必填，移动完成后的数组下标，范围 `0..路由数量-1` |
| revision | string | 必填，来自同一服务的列表快照 |

仅在指定服务内部移动，保留整条路由（包括 `group`、`terminal`、`@id` 和扩展字段），其他路由相对顺序不变；通过一次 `POST /load` 原子重载，不分两次交换。源下标不存在返回 404，目标位置无效返回 400，缺少版本返回 428，版本不符返回 409。成功后重新查询最新下标和版本。

Web 列表默认展示执行顺序，序号从 **1** 开始（API 下标从 **0** 开始）；支持同一服务内上移/下移。域名分组仅用于浏览，仍显示执行序号；分组或搜索状态下不提供调序操作。调序需要 `PUT /api/caddy/route/:index/position` 权限，并按写操作审计。

## 更新路由

```bash
isrvd_put "/caddy/route/<INDEX>" '{
  "match": [{"host": ["api.example.com"]}],
  "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "new-backend:8080"}]}]
}'
```

> 更新为整条路由替换，未提交的字段（`group`、`terminal`、`@id` 等）会被清除，应先 GET 再修改提交。创建、更新、删除均通过 `POST /load` 整体原子重载。

更新和删除均可通过 `?revision=<列表快照版本>` 启用版本校验（Web 界面会携带），列表变化返回 409；未携带时保留旧客户端行为。版本在 iSrvd 读取最新配置的修改临界区内检查，能够拒绝基于旧列表的操作；不代替其他 Caddy Admin API 客户端的并发写入协调。

## 删除路由

```bash
isrvd_delete "/caddy/route/<INDEX>"
```

> 注意：删除后下标会重排，后续操作请重新查询路由列表获取最新 index。
