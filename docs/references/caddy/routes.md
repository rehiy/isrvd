# Caddy 路由 API

> Caddy 路由模型与 APISIX 不同：路由是 `apps.http.servers.<name>.routes` 数组中的元素，
> 用数组下标（index）作为主键。默认 server 名称为 `srv0`，可通过 query 参数 `?server=<name>` 切换。

## 列出路由

```bash
isrvd_get "/caddy/routes"                      # 默认 server=srv0
isrvd_get "/caddy/routes?server=srv0"          # 显式指定 server
```

| 字段 | 类型 | 说明 |
|------|------|------|
| index | number | 数组下标（路由主键，只读，由后端返回） |
| group | string | 路由分组（可选） |
| match | object | 匹配条件，结构见下表 |
| handler | object | 处理器配置，结构见下表 |
| terminal | boolean | 终止后续路由匹配 |
| id | string | Caddy `@id` 字段（可选，用于 admin api 寻址） |

### match 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| hosts | string[] | 匹配 Host，留空表示不限制 |
| paths | string[] | 匹配 Path，支持 `*` 通配符 |
| methods | string[] | 匹配 HTTP 方法 |
| headers | object | 按请求头匹配，key 为头字段名，value 为匹配值列表；value 为空数组表示只要该头存在即匹配 |
| protocol | string | 匹配协议：`http` / `https`，留空不限制 |

> 字段为空数组等价于"不限制"。

### handler 字段

handler 通过 `kind` 字段区分类型，每种类型只填对应字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| kind | string | 处理器类型：`reverse_proxy` / `file_server` / `static_response` / `rewrite` / `headers` / `raw` |
| upstreams | string[] | 反向代理上游 `host:port` 列表（`kind=reverse_proxy`）；Web UI 可选择运行中 Docker 容器与端口自动填充 |
| dialTimeout | string | 建立 TCP 连接的超时，如 `10s`（`kind=reverse_proxy`，非 FastCGI 时有效） |
| readTimeout | string | 等待上游响应头的超时，如 `30s`（`kind=reverse_proxy`，非 FastCGI 时有效） |
| writeTimeout | string | 向上游写入请求的超时，如 `30s`（`kind=reverse_proxy`，非 FastCGI 时有效） |
| proxyRewrite | object | 转发前 URI 重写配置，写入 Caddy `reverse_proxy.rewrite`（`kind=reverse_proxy`） |
| proxyRewrite.method | string | 转发前改写请求方法，如 `GET` / `POST` |
| proxyRewrite.rewriteUri | string | 完整替换转发给上游的 URI，支持 Caddy 占位符 |
| proxyRewrite.stripPathPrefix | string | 转发前去掉路径前缀 |
| proxyRewrite.stripPathSuffix | string | 转发前去掉路径后缀 |
| proxyRewrite.uriSubstringFind | string | 转发前 URI 子串查找 |
| proxyRewrite.uriSubstringReplace | string | 转发前 URI 子串替换 |
| root | string | 静态文件根目录（`kind=file_server`） |
| browse | boolean | 是否开启目录浏览（`kind=file_server`） |
| statusCode | number | 响应状态码（`kind=static_response`） |
| body | string | 响应体内容（`kind=static_response`） |
| rewriteUri | string | 完整替换请求 URI，支持 Caddy 占位符（`kind=rewrite`） |
| stripPathPrefix | string | 去掉路径前缀（`kind=rewrite`） |
| stripPathSuffix | string | 去掉路径后缀（`kind=rewrite`） |
| uriSubstringFind | string | URI 子串查找（`kind=rewrite`） |
| uriSubstringReplace | string | URI 子串替换（`kind=rewrite`） |
| requestHeaders | HeaderOp[] | 请求头操作列表（`kind=headers`） |
| responseHeaders | HeaderOp[] | 响应头操作列表（`kind=headers`） |
| raw | any | 原始 handle 数组，任意 Caddy 模块，高级用法（`kind=raw`） |

#### HeaderOp 结构（`kind=headers` 使用）

| 字段 | 类型 | 说明 |
|------|------|------|
| op | string | 操作类型：`set`（覆盖写入）/ `add`（追加，允许多值）/ `delete`（删除该头） |
| field | string | 头字段名，如 `X-Real-IP`、`X-Frame-Options` |
| value | string | 值（`delete` 时留空）；支持 Caddy 占位符，如 `{http.request.remote.host}` |

## 查看路由详情

```bash
isrvd_get "/caddy/route/<INDEX>"
isrvd_get "/caddy/route/0?server=srv0"
```

## 创建路由

```bash
# 反向代理（含超时）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["api.example.com"], "paths": ["/v1/*"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["backend:8080"], "dialTimeout": "10s", "readTimeout": "30s"}
}'

# 按协议匹配（HTTP 跳转）
isrvd_post "/caddy/route" '{
  "match": {"protocol": "http"},
  "handler": {"kind": "static_response", "statusCode": 301, "body": ""},
  "terminal": true
}'

# 按请求头匹配
isrvd_post "/caddy/route" '{
  "match": {"headers": {"X-Internal": []}},
  "handler": {"kind": "reverse_proxy", "upstreams": ["internal-svc:8080"]}
}'

# 反向代理转发前重写 URI（生成 reverse_proxy.rewrite）
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/api/*"]},
  "handler": {
    "kind": "reverse_proxy",
    "upstreams": ["backend:8080"],
    "proxyRewrite": {"method": "GET", "stripPathPrefix": "/api"}
  }
}'

# 设置响应头（CORS / 安全头）
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/api/*"]},
  "handler": {
    "kind": "headers",
    "responseHeaders": [
      {"op": "set", "field": "Access-Control-Allow-Origin", "value": "*"},
      {"op": "set", "field": "X-Frame-Options", "value": "SAMEORIGIN"}
    ]
  }
}'

# 透传真实 IP（请求头）
isrvd_post "/caddy/route" '{
  "handler": {
    "kind": "headers",
    "requestHeaders": [
      {"op": "set", "field": "X-Real-IP", "value": "{http.request.remote.host}"},
      {"op": "set", "field": "X-Forwarded-For", "value": "{http.request.remote.host}"}
    ]
  }
}'

# 静态文件
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/static/*"]},
  "handler": {"kind": "file_server", "root": "/var/www", "browse": false}
}'

# 静态响应
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/health"]},
  "handler": {"kind": "static_response", "statusCode": 200, "body": "OK"}
}'

# URI 重写
 isrvd_post "/caddy/route" '{
  "match": {"paths": ["/api/*"]},
  "handler": {"kind": "rewrite", "stripPathPrefix": "/api"}
}'

# 完整 URI 替换（支持占位符）
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/old/*"]},
  "handler": {"kind": "rewrite", "rewriteUri": "/new/{http.request.uri.path.1}"}
}'

# 子串替换
isrvd_post "/caddy/route" '{
  "match": {"paths": ["/v1/*"]},
  "handler": {"kind": "rewrite", "uriSubstringFind": "/v1", "uriSubstringReplace": "/v2"}
}'

# 原始 handle 数组（如 headers + reverse_proxy 链式组合）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["a.example.com"]},
  "handler": {"kind": "raw", "raw": [
    {"handler": "headers", "response": {"set": {"X-Forwarded-Host": ["{http.request.host}"]}}},
    {"handler": "reverse_proxy", "upstreams": [{"dial": "backend:8080"}]}
  ]}
}'
```

返回值 `payload.index`：新路由在数组中的下标。

## 更新路由

```bash
isrvd_put "/caddy/route/<INDEX>" '{
  "match": {"hosts": ["api.example.com"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["new-backend:8080"]}
}'
```

> 路由更新会触发 Caddy 整体配置 reload（`POST /load`），原子替换。

## 删除路由

```bash
isrvd_delete "/caddy/route/<INDEX>"
```

> 注意：删除后下标会重排，后续操作请重新查询路由列表获取最新 index。
